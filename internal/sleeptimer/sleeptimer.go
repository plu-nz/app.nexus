// Package sleeptimer suspends the machine after a delay. It lives in the
// Nexus process, so the timer keeps running while the panel is hidden.
package sleeptimer

import (
	"sync"
	"time"
)

// State is what the panel shows about the timer.
type State struct {
	// Minutes is the duration that was chosen, or 0 when no timer runs.
	Minutes int `json:"minutes"`
	// Remaining is the time left, in whole minutes rounded up.
	Remaining int `json:"remaining"`
	// Error is why the last suspend failed, if it did.
	Error string `json:"error"`
}

// Timer runs one suspend countdown at a time.
type Timer struct {
	// Suspend puts the machine to sleep.
	Suspend func() error
	// unit is one minute; tests shorten it.
	unit time.Duration

	mu       sync.Mutex
	timer    *time.Timer
	minutes  int
	deadline time.Time
	err      string
}

// New returns a Timer that calls suspend when it fires.
func New(suspend func() error) *Timer {
	return &Timer{Suspend: suspend, unit: time.Minute}
}

// Start replaces any running countdown with one of the given minutes.
func (t *Timer) Start(minutes int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stop()
	t.minutes = minutes
	t.deadline = time.Now().Add(time.Duration(minutes) * t.unit)
	t.err = ""
	var fired *time.Timer
	fired = time.AfterFunc(time.Duration(minutes)*t.unit, func() {
		t.mu.Lock()
		if t.timer != fired {
			t.mu.Unlock()
			return
		}
		t.stop()
		t.mu.Unlock()
		err := t.Suspend()
		t.mu.Lock()
		if err != nil {
			t.err = err.Error()
		}
		t.mu.Unlock()
	})
	t.timer = fired
}

// Cancel stops the countdown.
func (t *Timer) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stop()
}

// SuspendNow stops any countdown and calls the suspend action immediately.
func (t *Timer) SuspendNow() error {
	t.Cancel()
	err := t.Suspend()
	t.mu.Lock()
	if err != nil {
		t.err = err.Error()
	}
	t.mu.Unlock()
	return err
}

func (t *Timer) stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = nil
	t.minutes = 0
	t.deadline = time.Time{}
}

// State reports the countdown.
func (t *Timer) State() State {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := State{Minutes: t.minutes, Error: t.err}
	if !t.deadline.IsZero() {
		left := time.Until(t.deadline)
		s.Remaining = int((left + t.unit - 1) / t.unit)
		if s.Remaining < 0 {
			s.Remaining = 0
		}
	}
	return s
}
