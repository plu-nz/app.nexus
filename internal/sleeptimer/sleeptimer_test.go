package sleeptimer

import (
	"errors"
	"testing"
	"time"
)

func fast(suspend func() error) *Timer {
	t := New(suspend)
	t.unit = 20 * time.Millisecond
	return t
}

func TestTimerSuspendsWhenItRunsOut(t *testing.T) {
	done := make(chan struct{})
	timer := fast(func() error { close(done); return nil })
	timer.Start(2)
	if s := timer.State(); s.Minutes != 2 || s.Remaining != 2 {
		t.Fatalf("State = %+v, want 2 of 2", s)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the timer never suspended")
	}
	if s := timer.State(); s != (State{}) {
		t.Fatalf("State after firing = %+v, want empty", s)
	}
}

func TestCancelStopsTheTimer(t *testing.T) {
	timer := fast(func() error { t.Error("suspended after cancel"); return nil })
	timer.Start(1)
	timer.Cancel()
	time.Sleep(60 * time.Millisecond)
	if s := timer.State(); s != (State{}) {
		t.Fatalf("State = %+v, want empty", s)
	}
}

func TestRestartReplacesTheTimer(t *testing.T) {
	calls := make(chan struct{}, 2)
	timer := fast(func() error { calls <- struct{}{}; return nil })
	timer.Start(1)
	timer.Start(3)
	time.Sleep(40 * time.Millisecond)
	if len(calls) != 0 {
		t.Fatal("the replaced timer fired")
	}
	if s := timer.State(); s.Minutes != 3 {
		t.Fatalf("State = %+v, want the new timer", s)
	}
	timer.Cancel()
}

func TestFailedSuspendIsReported(t *testing.T) {
	done := make(chan struct{})
	timer := fast(func() error { defer close(done); return errors.New("not allowed") })
	timer.Start(1)
	<-done
	time.Sleep(5 * time.Millisecond)
	if s := timer.State(); s.Error != "not allowed" {
		t.Fatalf("State = %+v, want the error", s)
	}
	timer.Start(1)
	if s := timer.State(); s.Error != "" {
		t.Fatalf("a new timer kept the old error: %+v", s)
	}
	timer.Cancel()
}

func TestSuspendNowCancelsCountdownAndReportsFailure(t *testing.T) {
	calls := 0
	timer := fast(func() error {
		calls++
		return errors.New("not allowed")
	})
	timer.Start(3)
	if err := timer.SuspendNow(); err == nil || err.Error() != "not allowed" {
		t.Fatalf("SuspendNow error = %v, want not allowed", err)
	}
	time.Sleep(80 * time.Millisecond)
	if calls != 1 {
		t.Fatalf("suspend calls = %d, want 1", calls)
	}
	if s := timer.State(); s.Minutes != 0 || s.Error != "not allowed" {
		t.Fatalf("State = %+v, want canceled timer and error", s)
	}
}
