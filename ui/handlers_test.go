package ui

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
	"github.com/jmoiron/sqlx"
)

// connects to a database in a temporary home directory,
// so the tests never touch the user's sessions
func testDatabase(t *testing.T) *sqlx.DB {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	database, err := db.Connect()
	if err != nil {
		t.Fatalf("failed to connect to the test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return database
}

func TestRecordSessionStoresStartTime(t *testing.T) {
	database := testDatabase(t)

	// a 25 minute session that started 25 minutes ago
	start := time.Now().Add(-25 * time.Minute).Truncate(time.Second)
	m := Model{
		repo:            db.NewSessionRepo(database),
		currentTaskType: config.WorkTask,
		sessionStart:    start,
		elapsed:         25 * time.Minute,
	}

	m.recordSession()

	var startedAt string
	if err := database.Get(&startedAt, "select started_at from sessions"); err != nil {
		t.Fatalf("failed to read the recorded session: %v", err)
	}

	got, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		t.Fatalf("started_at %q is not RFC 3339: %v", startedAt, err)
	}

	if !got.Equal(start) {
		t.Errorf("started_at = %v, want the session start %v", got, start)
	}
}

func TestStartSessionSetsStartTime(t *testing.T) {
	m := Model{sessionStart: time.Now().Add(-time.Hour)}

	before := time.Now()
	m.startSession(config.BreakTask, config.Task{Duration: 5 * time.Minute}, false)

	if m.sessionStart.Before(before) {
		t.Errorf("sessionStart = %v, want a time after %v", m.sessionStart, before)
	}
}

func TestLongBreakNext(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		taskType config.TaskType
		position int
		want     bool
	}{
		{"last work session of the cycle", true, config.WorkTask, 4, true},
		{"earlier work session", true, config.WorkTask, 3, false},
		{"break session", true, config.BreakTask, 4, false},
		{"long breaks disabled", false, config.WorkTask, 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				longBreak:       config.LongBreak{Enabled: tt.enabled, After: 4},
				currentTaskType: tt.taskType,
				cyclePosition:   tt.position,
			}

			if got := m.longBreakNext(); got != tt.want {
				t.Errorf("longBreakNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndedSession(t *testing.T) {
	start := time.Now().Add(-25 * time.Minute)
	m := Model{
		longBreak:       config.LongBreak{Enabled: true, After: 4},
		currentTaskType: config.WorkTask,
		currentTask:     config.Task{Title: "write report", Duration: 25 * time.Minute},
		cyclePosition:   4,
		sessionStart:    start,
		duration:        26 * time.Minute, // a minute was added during the session
		elapsed:         26 * time.Minute,
	}

	got := m.endedSession()

	if got.Type != "work" || got.Title != "write report" || !got.LongBreakNext {
		t.Errorf("endedSession() = %+v, want a work session titled %q before a long break", got, "write report")
	}

	if got.Duration != 26*time.Minute || got.Elapsed != 26*time.Minute || !got.StartedAt.Equal(start) {
		t.Errorf("endedSession() = %+v, want 26m planned and elapsed, started at %v", got, start)
	}
}
