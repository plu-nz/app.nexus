package store

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/plu-nz/app.nexus/internal/apps"
	"github.com/plu-nz/app.nexus/internal/flatpak"
)

var catalog = []apps.App{
	{ID: "lounge", Name: "Lounge", Command: "lounge", Source: apps.External, DesktopID: "nz.plu.lounge"},
	{ID: "kodi", Name: "Kodi", Command: "flatpak:tv.kodi.Kodi", Source: apps.Flathub, DesktopID: "tv.kodi.Kodi"},
	{ID: "moonlight", Name: "Moonlight", Command: "flatpak:com.moonlight_stream.Moonlight", Source: apps.Flathub, DesktopID: "com.moonlight_stream.Moonlight"},
	{ID: "steam", Name: "Steam", Command: "flatpak:com.valvesoftware.Steam", Source: apps.Flathub, DesktopID: "com.valvesoftware.Steam"},
}

// fakeFlatpak records operations. An operation blocks while hold has a
// channel for its app ID, until that channel closes or the context ends.
type fakeFlatpak struct {
	mu         sync.Mutex
	installed  map[string]flatpak.Installation
	updates    map[string]bool
	updatesErr error
	fail       map[string]error
	hold       map[string]chan struct{}
	ops        []string
}

func (f *fakeFlatpak) Installed(context.Context) (map[string]flatpak.Installation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := map[string]flatpak.Installation{}
	for k, v := range f.installed {
		copied[k] = v
	}
	return copied, nil
}

func (f *fakeFlatpak) Updates(context.Context) (map[string]bool, error) {
	return f.updates, f.updatesErr
}

func (f *fakeFlatpak) Info(_ context.Context, id string) (flatpak.Info, error) {
	return flatpak.Info{DownloadSize: 1, Runtime: id}, nil
}

func (f *fakeFlatpak) op(ctx context.Context, name, id string, progress func(flatpak.Step), apply func()) error {
	f.mu.Lock()
	f.ops = append(f.ops, name+" "+id)
	wait, fail := f.hold[id], f.fail[id]
	f.mu.Unlock()
	if progress != nil {
		progress(flatpak.Step{Action: name, Kind: "app", ID: id, N: 1})
	}
	if wait != nil {
		select {
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if fail != nil {
		return fail
	}
	f.mu.Lock()
	apply()
	f.mu.Unlock()
	return nil
}

func (f *fakeFlatpak) Install(ctx context.Context, id string, p func(flatpak.Step)) error {
	return f.op(ctx, "install", id, p, func() { f.installed[id] = flatpak.User })
}

func (f *fakeFlatpak) Update(ctx context.Context, id string, p func(flatpak.Step)) error {
	return f.op(ctx, "update", id, p, func() {})
}

func (f *fakeFlatpak) Uninstall(ctx context.Context, id string, p func(flatpak.Step)) error {
	return f.op(ctx, "uninstall", id, p, func() { delete(f.installed, id) })
}

type recorder struct {
	mu      sync.Mutex
	jobs    []Job
	changed int
	ended   chan Job
}

func newStore(f *fakeFlatpak) (*Store, *recorder) {
	r := &recorder{ended: make(chan Job, 16)}
	s := &Store{
		Catalog: catalog,
		Flatpak: f,
		Icon:    func(id string) string { return "icon:" + id },
		Notify: func(j Job) {
			r.mu.Lock()
			r.jobs = append(r.jobs, j)
			r.mu.Unlock()
			if j.State == Done || j.State == Failed || j.State == Canceled {
				r.ended <- j
			}
		},
		Changed: func() {
			r.mu.Lock()
			r.changed++
			r.mu.Unlock()
		},
	}
	return s, r
}

func (r *recorder) next(t *testing.T) Job {
	t.Helper()
	select {
	case j := <-r.ended:
		return j
	case <-time.After(3 * time.Second):
		t.Fatal("no job ended")
		return Job{}
	}
}

func newFake() *fakeFlatpak {
	return &fakeFlatpak{
		installed: map[string]flatpak.Installation{},
		fail:      map[string]error{},
		hold:      map[string]chan struct{}{},
	}
}

func TestItemsReportStates(t *testing.T) {
	f := newFake()
	f.installed = map[string]flatpak.Installation{"tv.kodi.Kodi": flatpak.User, "com.valvesoftware.Steam": flatpak.System}
	f.updates = map[string]bool{"tv.kodi.Kodi": true, "com.valvesoftware.Steam": true}
	s, _ := newStore(f)
	items, err := s.Items(context.Background(), func(c string) bool { return c == "lounge" })
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][3]bool{}
	for _, it := range items {
		got[it.ID] = [3]bool{it.Installed, it.System, it.Update}
		if it.Installed != (it.Icon != "") {
			t.Errorf("%s: icon %q for installed=%v", it.ID, it.Icon, it.Installed)
		}
	}
	want := map[string][3]bool{
		"lounge":    {true, false, false},
		"kodi":      {true, false, true},
		"moonlight": {false, false, false},
		// An update for a system install is not ours to apply.
		"steam": {true, true, false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("states %v, want %v", got, want)
	}
}

func TestItemsWorkOfflineAndWithoutFlatpak(t *testing.T) {
	f := newFake()
	f.updatesErr = errors.New("offline")
	s, _ := newStore(f)
	if _, err := s.Items(context.Background(), func(string) bool { return false }); err != nil {
		t.Fatalf("Items failed offline: %v", err)
	}
	s.Flatpak = nil
	items, err := s.Items(context.Background(), func(c string) bool { return c == "lounge" })
	if err != nil || len(items) != len(catalog) || !items[0].Installed || items[1].Installed {
		t.Fatalf("Items without flatpak = %+v, %v", items, err)
	}
	if err := s.Start(context.Background(), "kodi", Install); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Start without flatpak = %v", err)
	}
}

func TestStartRejectsWhatItCannotDo(t *testing.T) {
	f := newFake()
	f.installed = map[string]flatpak.Installation{"tv.kodi.Kodi": flatpak.User, "com.valvesoftware.Steam": flatpak.System}
	s, _ := newStore(f)
	ctx := context.Background()
	cases := []struct {
		app    string
		action Action
		want   error
	}{
		{"nope", Install, ErrNotInCatalog},
		{"lounge", Install, ErrNotFromFlathub},
		{"kodi", Install, ErrState},
		{"moonlight", Remove, ErrState},
		{"moonlight", Update, ErrState},
		{"steam", Remove, ErrState},
		{"steam", Update, ErrState},
		{"kodi", "launch", ErrState},
	}
	for _, c := range cases {
		if err := s.Start(ctx, c.app, c.action); !errors.Is(err, c.want) {
			t.Errorf("Start(%s, %s) = %v, want %v", c.app, c.action, err, c.want)
		}
	}
	if len(f.ops) != 0 {
		t.Fatalf("ran %v", f.ops)
	}
}

func TestJobsRunInOrderOneAtATime(t *testing.T) {
	f := newFake()
	gate := make(chan struct{})
	f.hold["tv.kodi.Kodi"] = gate
	s, r := newStore(f)
	ctx := context.Background()
	if err := s.Start(ctx, "kodi", Install); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(ctx, "moonlight", Install); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(ctx, "kodi", Install); !errors.Is(err, ErrBusy) {
		t.Fatalf("second Kodi job = %v, want ErrBusy", err)
	}
	waitFor(t, func() bool { return len(s.Jobs()) == 2 && s.Jobs()[0].State == Running })
	if jobs := s.Jobs(); jobs[1].App != "moonlight" || jobs[1].State != Queued {
		t.Fatalf("jobs %+v", jobs)
	}
	close(gate)
	first, second := r.next(t), r.next(t)
	if first.App != "kodi" || first.State != Done || second.App != "moonlight" || second.State != Done {
		t.Fatalf("ended %+v then %+v", first, second)
	}
	if want := []string{"install tv.kodi.Kodi", "install com.moonlight_stream.Moonlight"}; !reflect.DeepEqual(f.ops, want) {
		t.Fatalf("ops %v, want %v", f.ops, want)
	}
	if r.changed != 2 || len(s.Jobs()) != 0 {
		t.Fatalf("changed %d times, %d jobs left", r.changed, len(s.Jobs()))
	}
	// Progress reached the UI with the running state.
	sawStep := false
	for _, j := range r.jobs {
		if j.App == "kodi" && j.State == Running && j.Step != nil && j.Step.ID == "tv.kodi.Kodi" {
			sawStep = true
		}
	}
	if !sawStep {
		t.Fatalf("no progress step in %+v", r.jobs)
	}
}

func TestCancelRunningAndQueuedJobs(t *testing.T) {
	f := newFake()
	f.hold["tv.kodi.Kodi"] = make(chan struct{})
	s, r := newStore(f)
	ctx := context.Background()
	_ = s.Start(ctx, "kodi", Install)
	_ = s.Start(ctx, "moonlight", Install)
	waitFor(t, func() bool { return len(s.Jobs()) > 0 && s.Jobs()[0].State == Running })

	s.Cancel("moonlight")
	if j := r.next(t); j.App != "moonlight" || j.State != Canceled {
		t.Fatalf("queued cancel ended %+v", j)
	}
	s.Cancel("kodi")
	if j := r.next(t); j.App != "kodi" || j.State != Canceled {
		t.Fatalf("running cancel ended %+v", j)
	}
	if want := []string{"install tv.kodi.Kodi"}; !reflect.DeepEqual(f.ops, want) {
		t.Fatalf("ops %v, want only Kodi", f.ops)
	}
	if _, ok := f.installed["tv.kodi.Kodi"]; ok {
		t.Fatal("a canceled install is installed")
	}
}

func TestFailedJobKeepsTheError(t *testing.T) {
	f := newFake()
	f.installed = map[string]flatpak.Installation{"tv.kodi.Kodi": flatpak.User}
	f.fail["tv.kodi.Kodi"] = errors.New("Unable to connect to dl.flathub.org")
	s, r := newStore(f)
	if err := s.Start(context.Background(), "kodi", Update); err != nil {
		t.Fatal(err)
	}
	if j := r.next(t); j.State != Failed || j.Error != "Unable to connect to dl.flathub.org" {
		t.Fatalf("ended %+v", j)
	}
}

func TestRemoveThenInstallAgain(t *testing.T) {
	f := newFake()
	f.installed = map[string]flatpak.Installation{"tv.kodi.Kodi": flatpak.User}
	s, r := newStore(f)
	ctx := context.Background()
	if err := s.Start(ctx, "kodi", Remove); err != nil {
		t.Fatal(err)
	}
	if j := r.next(t); j.State != Done {
		t.Fatalf("remove ended %+v", j)
	}
	if err := s.Start(ctx, "kodi", Install); err != nil {
		t.Fatalf("install after remove = %v", err)
	}
	if j := r.next(t); j.State != Done {
		t.Fatalf("install ended %+v", j)
	}
}

func TestInfoOnlyForFlathubApps(t *testing.T) {
	s, _ := newStore(newFake())
	if info, err := s.Info(context.Background(), "kodi"); err != nil || info.Runtime != "tv.kodi.Kodi" {
		t.Fatalf("Info(kodi) = %+v, %v", info, err)
	}
	if _, err := s.Info(context.Background(), "lounge"); !errors.Is(err, ErrNotFromFlathub) {
		t.Fatalf("Info(lounge) = %v", err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
