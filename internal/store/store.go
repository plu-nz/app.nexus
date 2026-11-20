// Package store installs, updates, and removes catalog apps from Flathub,
// one operation at a time. Only catalog apps can be changed, and only in
// the user's own Flatpak installation.
package store

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/plu-nz/app.nexus/internal/apps"
	"github.com/plu-nz/app.nexus/internal/flatpak"
)

// Flatpak is what the store needs from the flatpak command.
type Flatpak interface {
	Installed(ctx context.Context) (map[string]flatpak.Installation, error)
	Updates(ctx context.Context) (map[string]bool, error)
	Info(ctx context.Context, id string) (flatpak.Info, error)
	Install(ctx context.Context, id string, progress func(flatpak.Step)) error
	Update(ctx context.Context, id string, progress func(flatpak.Step)) error
	Uninstall(ctx context.Context, id string, progress func(flatpak.Step)) error
}

// Item is a catalog app with its install state.
type Item struct {
	apps.App
	Installed bool `json:"installed"`
	// System marks an app installed for all users: Nexus can open it but
	// not update or remove it, because that needs administrator rights.
	System bool `json:"system"`
	Update bool `json:"update"`
}

// Action is a change to an app.
type Action string

const (
	Install Action = "install"
	Update  Action = "update"
	Remove  Action = "remove"
)

// Job states, as the UI receives them.
const (
	Queued   = "queued"
	Running  = "running"
	Done     = "done"
	Failed   = "failed"
	Canceled = "canceled"
)

// Job is an operation on one app. Every change to it is sent to Notify.
type Job struct {
	App    string        `json:"app"`
	Action Action        `json:"action"`
	State  string        `json:"state"`
	Step   *flatpak.Step `json:"step,omitempty"`
	Error  string        `json:"error,omitempty"`
}

var (
	// ErrUnavailable means flatpak is not installed.
	ErrUnavailable = errors.New("flatpak is not installed")
	// ErrNotInCatalog rejects apps outside the catalog.
	ErrNotInCatalog = errors.New("app is not in the catalog")
	// ErrNotFromFlathub rejects apps with their own installer.
	ErrNotFromFlathub = errors.New("app is not installed from Flathub")
	// ErrState rejects an action the app's state does not allow, such as
	// removing an app that is not installed or one installed for all users.
	ErrState = errors.New("app cannot do that now")
	// ErrBusy rejects a second operation on an app that already has one.
	ErrBusy = errors.New("app already has an operation")
)

// Store runs operations on catalog apps.
type Store struct {
	Catalog []apps.App
	// Flatpak is nil when flatpak is not installed.
	Flatpak Flatpak
	// Icon returns an app's icon as a data URL, or "".
	Icon func(desktopID string) string
	// Notify receives every job change. It must not block.
	Notify func(Job)
	// Changed runs after an operation changed what is installed.
	Changed func()

	mu      sync.Mutex
	jobs    []*Job // the running job first, then the queue
	cancel  context.CancelFunc
	working bool
}

// Available reports whether Flatpak apps can be installed.
func (s *Store) Available() bool { return s.Flatpak != nil }

// Items returns the catalog with each app's state. Without flatpak, only
// apps with their own installer can show as installed.
func (s *Store) Items(ctx context.Context, installedCommand func(string) bool) ([]Item, error) {
	var installed map[string]flatpak.Installation
	var updates map[string]bool
	if s.Flatpak != nil {
		var err error
		if installed, err = s.Flatpak.Installed(ctx); err != nil {
			return nil, err
		}
		// Updates needs the network; without it, the store still works.
		if updates, err = s.Flatpak.Updates(ctx); err != nil {
			log.Printf("checking for app updates: %v", err)
		}
	}
	items := make([]Item, 0, len(s.Catalog))
	for _, a := range s.Catalog {
		item := Item{App: a}
		if id := a.FlatpakID(); id != "" {
			where, ok := installed[id]
			item.Installed = ok
			item.System = where == flatpak.System
			item.Update = where == flatpak.User && updates[id]
		} else {
			item.Installed = installedCommand(a.Command)
		}
		if item.Installed && s.Icon != nil {
			item.Icon = s.Icon(a.DesktopID)
		}
		items = append(items, item)
	}
	return items, nil
}

// Info returns Flathub's sizes for a catalog app.
func (s *Store) Info(ctx context.Context, appID string) (flatpak.Info, error) {
	a, err := s.flathubApp(appID)
	if err != nil {
		return flatpak.Info{}, err
	}
	return s.Flatpak.Info(ctx, a.FlatpakID())
}

// Jobs returns the running and queued jobs.
func (s *Store) Jobs() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]Job, len(s.jobs))
	for i, j := range s.jobs {
		jobs[i] = *j
	}
	return jobs
}

// Start queues an action on a catalog app after checking that its state
// allows it.
func (s *Store) Start(ctx context.Context, appID string, action Action) error {
	a, err := s.flathubApp(appID)
	if err != nil {
		return err
	}
	installed, err := s.Flatpak.Installed(ctx)
	if err != nil {
		return err
	}
	where, ok := installed[a.FlatpakID()]
	switch action {
	case Install:
		if ok {
			return ErrState
		}
	case Update, Remove:
		if where != flatpak.User {
			return ErrState
		}
	default:
		return ErrState
	}

	s.mu.Lock()
	for _, j := range s.jobs {
		if j.App == appID {
			s.mu.Unlock()
			return ErrBusy
		}
	}
	job := &Job{App: appID, Action: action, State: Queued}
	s.jobs = append(s.jobs, job)
	start := !s.working
	s.working = true
	s.mu.Unlock()

	s.notify(job)
	if start {
		go s.work()
	}
	return nil
}

// Cancel stops an app's running job or drops it from the queue.
func (s *Store) Cancel(appID string) {
	s.mu.Lock()
	for i, j := range s.jobs {
		if j.App != appID {
			continue
		}
		if i == 0 && j.State == Running {
			if s.cancel != nil {
				s.cancel()
			}
			s.mu.Unlock()
			return
		}
		s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
		j.State = Canceled
		s.mu.Unlock()
		s.notify(j)
		return
	}
	s.mu.Unlock()
}

// work runs queued jobs until the queue is empty.
func (s *Store) work() {
	for {
		s.mu.Lock()
		if len(s.jobs) == 0 {
			s.working = false
			s.mu.Unlock()
			return
		}
		job := s.jobs[0]
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		job.State = Running
		s.mu.Unlock()
		s.notify(job)

		err := s.run(ctx, job)
		cancel()

		s.mu.Lock()
		s.cancel = nil
		s.jobs = s.jobs[1:]
		switch {
		case errors.Is(err, context.Canceled):
			job.State = Canceled
		case err != nil:
			job.State, job.Error = Failed, err.Error()
			log.Printf("%s %s: %v", job.Action, job.App, err)
		default:
			job.State = Done
		}
		s.mu.Unlock()
		// A canceled install can still have left runtimes behind; the
		// installed list may have changed either way.
		if s.Changed != nil {
			s.Changed()
		}
		s.notify(job)
	}
}

func (s *Store) run(ctx context.Context, job *Job) error {
	a, ok := s.find(job.App)
	if !ok {
		return ErrNotInCatalog
	}
	progress := func(step flatpak.Step) {
		s.mu.Lock()
		job.Step = &step
		s.mu.Unlock()
		s.notify(job)
	}
	switch job.Action {
	case Install:
		return s.Flatpak.Install(ctx, a.FlatpakID(), progress)
	case Update:
		return s.Flatpak.Update(ctx, a.FlatpakID(), progress)
	case Remove:
		return s.Flatpak.Uninstall(ctx, a.FlatpakID(), progress)
	}
	return ErrState
}

func (s *Store) notify(job *Job) {
	if s.Notify == nil {
		return
	}
	s.mu.Lock()
	copied := *job
	if job.Step != nil {
		step := *job.Step
		copied.Step = &step
	}
	s.mu.Unlock()
	s.Notify(copied)
}

func (s *Store) find(appID string) (apps.App, bool) {
	for _, a := range s.Catalog {
		if a.ID == appID {
			return a, true
		}
	}
	return apps.App{}, false
}

func (s *Store) flathubApp(appID string) (apps.App, error) {
	if s.Flatpak == nil {
		return apps.App{}, ErrUnavailable
	}
	a, ok := s.find(appID)
	if !ok {
		return apps.App{}, ErrNotInCatalog
	}
	if a.Source != apps.Flathub || a.FlatpakID() == "" {
		return apps.App{}, ErrNotFromFlathub
	}
	return a, nil
}
