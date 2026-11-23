package actions

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
)

var endedSession = Session{
	Type:          "work",
	Title:         "write report",
	Duration:      25 * time.Minute,
	Elapsed:       24*time.Minute + 30*time.Second,
	StartedAt:     time.Date(2026, 11, 23, 9, 0, 0, 0, time.FixedZone("EST", -5*3600)),
	LongBreakNext: true,
}

func TestSessionEnviron(t *testing.T) {
	want := []string{
		"POMO_TYPE=work",
		"POMO_TITLE=write report",
		"POMO_DURATION=1500",
		"POMO_ELAPSED=1470",
		"POMO_STARTED_AT=2026-11-23T09:00:00-05:00",
		"POMO_LONG_BREAK=false",
		"POMO_LONG_BREAK_NEXT=true",
	}

	if got := endedSession.Environ(); !slices.Equal(got, want) {
		t.Errorf("Environ() = %q, want %q", got, want)
	}
}

func TestSessionEnvironNoLongBreak(t *testing.T) {
	session := endedSession
	session.LongBreakNext = false

	if got := session.Environ(); !slices.Contains(got, "POMO_LONG_BREAK_NEXT=false") {
		t.Errorf("Environ() = %q, want POMO_LONG_BREAK_NEXT=false", got)
	}
}

func TestPostCommandsReceiveSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}

	out := filepath.Join(t.TempDir(), "env")
	script := `printf '%s|%s|%s' "$POMO_TYPE" "$POMO_TITLE" "$POMO_LONG_BREAK_NEXT" > "$1"`

	runCommands(context.Background(), [][]string{{"sh", "-c", script, "sh", out}}, endedSession)

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the post command did not run: %v", err)
	}

	if want := "work|write report|true"; string(got) != want {
		t.Errorf("post command saw %q, want %q", got, want)
	}
}

func TestPostCommandsKeepEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}

	t.Setenv("POMO_TEST_INHERITED", "kept")
	out := filepath.Join(t.TempDir(), "env")
	script := `printf '%s' "$POMO_TEST_INHERITED" > "$1"`

	runCommands(context.Background(), [][]string{{"sh", "-c", script, "sh", out}}, endedSession)

	got, _ := os.ReadFile(out)
	if strings.TrimSpace(string(got)) != "kept" {
		t.Errorf("post command saw %q, want the inherited environment", got)
	}
}

func TestSessionEnvironLongBreak(t *testing.T) {
	session := Session{Type: "break", LongBreak: true}

	if got := session.Environ(); !slices.Contains(got, "POMO_LONG_BREAK=true") {
		t.Errorf("Environ() = %q, want POMO_LONG_BREAK=true", got)
	}
}

func TestStartActionsRunStartCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}

	out := filepath.Join(t.TempDir(), "env")
	script := `printf '%s|%s' "$POMO_TYPE" "$POMO_LONG_BREAK" > "$1"`
	task := config.Task{
		Start: [][]string{{"sh", "-c", script, "sh", out}},
		// must not run when a session starts
		Then: [][]string{{"sh", "-c", `printf 'then ran' > "$1"`, "sh", out}},
	}

	RunStartActions(context.Background(), task, Session{Type: "break", LongBreak: true}).Wait()

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the start command did not run: %v", err)
	}

	if want := "break|true"; string(got) != want {
		t.Errorf("start command saw %q, want %q", got, want)
	}
}
