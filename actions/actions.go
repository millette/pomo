// Package actions provides functionality to run commands when a session starts
// and post actions after it is completed.
package actions

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/gen2brain/beeep"
)

var CommandTimeout = 5 * time.Second

// Session describes the session that just started or ended.
// Start and post commands receive it through their environment.
type Session struct {
	Type          string        // "work" or "break"
	Title         string        // title shown in the timer
	Duration      time.Duration // planned duration, including any time added
	Elapsed       time.Duration // time the timer actually ran
	StartedAt     time.Time     // when the session started
	LongBreak     bool          // the session is a long break
	LongBreakNext bool          // a long break follows this work session
}

// Environ returns the session as environment variables,
// in the form "key=value".
//
// durations are in whole seconds
func (s Session) Environ() []string {
	seconds := func(d time.Duration) string {
		return strconv.Itoa(int(d.Seconds()))
	}

	return []string{
		"POMO_TYPE=" + s.Type,
		"POMO_TITLE=" + s.Title,
		"POMO_DURATION=" + seconds(s.Duration),
		"POMO_ELAPSED=" + seconds(s.Elapsed),
		"POMO_STARTED_AT=" + s.StartedAt.Format(time.RFC3339),
		"POMO_LONG_BREAK=" + strconv.FormatBool(s.LongBreak),
		"POMO_LONG_BREAK_NEXT=" + strconv.FormatBool(s.LongBreakNext),
	}
}

// RunStartActions runs the start commands of a session that just started,
// using a goroutine
//
// returns a wait group to wait for their completion
func RunStartActions(ctx context.Context, task config.Task, session Session) *sync.WaitGroup {
	var wg sync.WaitGroup

	wg.Go(func() {
		runCommands(ctx, task.Start, session)
	})

	return &wg
}

// RunPostActions sends task notification and runs post commands using goroutines
//
// returns a wait group to wait for their completion
func RunPostActions(ctx context.Context, task config.Task, session Session) *sync.WaitGroup {
	var wg sync.WaitGroup

	wg.Go(func() {
		sendNotification(task.Notification)
	})

	wg.Go(func() {
		runCommands(ctx, task.Then, session)
	})

	return &wg
}

// sends a notification using the beeep package
func sendNotification(notification config.Notification) {
	if !notification.Enabled {
		log.Println("notification disabled")
		return
	}

	log.Println("sending notification")

	// use the embedded icon
	var icon any = config.Icon

	// if the user has specified an icon
	// use that instead
	if len(notification.Icon) > 0 {
		icon = notification.Icon
	}

	var err error
	if notification.Urgent {
		err = beeep.Alert(notification.Title, notification.Message, icon)
	} else {
		err = beeep.Notify(notification.Title, notification.Message, icon)
	}

	if err != nil {
		log.Println("failed to send notification:", err)
	}
}

// runs the commands of a task, one after the other,
// with the session added to their environment
func runCommands(ctx context.Context, cmds [][]string, session Session) {
	log.Println("running commands")

	env := append(os.Environ(), session.Environ()...)

	for _, cmd := range cmds {
		c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
		c.Env = env

		if err := c.Run(); err != nil {
			// TODO: show error message
			log.Printf("failed to run command '%q': %v\n", cmd, err)
		}
	}
}
