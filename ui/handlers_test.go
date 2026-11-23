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
