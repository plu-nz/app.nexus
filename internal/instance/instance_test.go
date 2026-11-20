package instance

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSocketPathFallsBackWhenTooLong(t *testing.T) {
	if got := socketPath("/run/user/1000", "/tmp", 1000); got != "/run/user/1000/plunz-nexus-1000.sock" {
		t.Fatalf("socketPath = %q", got)
	}
	long := "/" + strings.Repeat("x", 120)
	if got := socketPath(long, "/tmp", 1000); got != "/tmp/plunz-nexus-1000.sock" {
		t.Fatalf("socketPath = %q, want the temp fallback", got)
	}
	if got := socketPath("", "/tmp", 7); got != "/tmp/plunz-nexus-7.sock" {
		t.Fatalf("socketPath = %q, want the temp fallback", got)
	}
}

func TestToggleReachesTheListener(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.sock")
	if err := Toggle(path); err == nil {
		t.Fatal("Toggle succeeded with nobody listening")
	}
	got := make(chan struct{}, 1)
	stop, err := Listen(path, func() { got <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if err := Toggle(path); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("the listener never heard the toggle")
	}
}
