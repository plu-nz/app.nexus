// Package flatpak installs, updates, and removes Flathub apps for the
// current user through the flatpak command. It never needs root: apps go
// into the user installation, and apps installed for all users are only
// reported, never changed.
package flatpak

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// FlathubURL is the repository file Nexus adds when the user installation
// has no flathub remote.
const FlathubURL = "https://dl.flathub.org/repo/flathub.flatpakrepo"

// Installation says where an app is installed.
type Installation string

const (
	User   Installation = "user"
	System Installation = "system"
)

// Step is one line of progress: flatpak installs an app as a series of
// refs (runtimes and extensions first, the app last) and reports each one
// as it starts. It reports no percentage when it does not run in a
// terminal.
type Step struct {
	// Action is "install", "update", or "uninstall".
	Action string `json:"action"`
	// Kind is "app" or "runtime".
	Kind string `json:"kind"`
	// ID is the ref's ID, such as org.freedesktop.Platform.
	ID string `json:"id"`
	// N counts the steps so far, from 1.
	N int `json:"n"`
}

// Info is what Flathub says about an app before it is installed.
type Info struct {
	// DownloadSize and InstalledSize are in bytes, for the app alone.
	DownloadSize  int64 `json:"downloadSize"`
	InstalledSize int64 `json:"installedSize"`
	// Runtime is the runtime ref, such as org.freedesktop.Platform/x86_64/26.08.
	Runtime string `json:"runtime"`
	// NeedsRuntime reports that the runtime is not installed yet, so the
	// install also downloads it.
	NeedsRuntime bool `json:"needsRuntime"`
}

// Flatpak runs the flatpak command. The zero value is not usable; use New.
// Tests replace Output and Stream.
type Flatpak struct {
	// Output runs flatpak with the arguments and returns its standard
	// output.
	Output func(ctx context.Context, args ...string) ([]byte, error)
	// Stream runs flatpak with the arguments and calls line for each line
	// it prints, standard error included.
	Stream func(ctx context.Context, line func(string), args ...string) error
}

// New returns a Flatpak that runs the real command, or nil when flatpak is
// not installed.
func New() *Flatpak {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return nil
	}
	return &Flatpak{Output: output, Stream: stream}
}

// Installed returns the installed apps and where each is installed. An
// app installed in both places counts as a user install, the one Nexus
// can change.
func (f *Flatpak) Installed(ctx context.Context) (map[string]Installation, error) {
	out, err := f.Output(ctx, "list", "--app", "--columns=application,installation")
	if err != nil {
		return nil, err
	}
	found := map[string]Installation{}
	for _, line := range lines(out) {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		where := Installation(fields[1])
		if where != User && where != System {
			continue
		}
		if found[fields[0]] != User {
			found[fields[0]] = where
		}
	}
	return found, nil
}

// Updates returns the user-installed apps that have an update.
func (f *Flatpak) Updates(ctx context.Context) (map[string]bool, error) {
	out, err := f.Output(ctx, "remote-ls", "--user", "--updates", "--app", "--columns=application")
	if err != nil {
		return nil, err
	}
	found := map[string]bool{}
	for _, line := range lines(out) {
		found[strings.TrimSpace(line)] = true
	}
	return found, nil
}

// Info asks Flathub about an app.
func (f *Flatpak) Info(ctx context.Context, id string) (Info, error) {
	out, err := f.Output(ctx, "remote-info", "--user", "flathub", id)
	if err != nil {
		return Info{}, err
	}
	var info Info
	for _, line := range lines(out) {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "Download Size", "Download":
			info.DownloadSize = parseSize(value)
		case "Installed Size", "Installed":
			info.InstalledSize = parseSize(value)
		case "Runtime":
			info.Runtime = value
		}
	}
	if info.Runtime != "" {
		runtimes, err := f.Output(ctx, "list", "--runtime", "--columns=ref")
		if err != nil {
			return Info{}, err
		}
		info.NeedsRuntime = true
		for _, line := range lines(runtimes) {
			if strings.TrimSpace(line) == info.Runtime {
				info.NeedsRuntime = false
			}
		}
	}
	return info, nil
}

// Install installs a Flathub app for the current user, adding the flathub
// remote first when the user installation has none.
func (f *Flatpak) Install(ctx context.Context, id string, progress func(Step)) error {
	if _, err := f.Output(ctx, "remote-add", "--user", "--if-not-exists", "flathub", FlathubURL); err != nil {
		return fmt.Errorf("adding Flathub: %w", err)
	}
	return f.run(ctx, progress, "install", "--user", "--noninteractive", "-y", "flathub", id)
}

// Update updates a user-installed app and what it depends on.
func (f *Flatpak) Update(ctx context.Context, id string, progress func(Step)) error {
	return f.run(ctx, progress, "update", "--user", "--noninteractive", "-y", id)
}

// Uninstall removes a user-installed app. Its data in ~/.var/app stays,
// so reinstalling it keeps the user's settings and library.
func (f *Flatpak) Uninstall(ctx context.Context, id string, progress func(Step)) error {
	return f.run(ctx, progress, "uninstall", "--user", "--noninteractive", "-y", id)
}

// run streams a flatpak operation, turning its lines into steps. A
// failure returns flatpak's own error message.
func (f *Flatpak) run(ctx context.Context, progress func(Step), args ...string) error {
	n := 0
	var lastError string
	err := f.Stream(ctx, func(line string) {
		if step, ok := ParseStep(line); ok {
			n++
			step.N = n
			if progress != nil {
				progress(step)
			}
			return
		}
		if msg, ok := strings.CutPrefix(strings.TrimSpace(line), "error: "); ok {
			lastError = msg
		}
	}, args...)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if lastError != "" {
		return errors.New(lastError)
	}
	return err
}

// verbs maps flatpak's progress verbs to step actions.
var verbs = map[string]string{"Installing": "install", "Updating": "update", "Uninstalling": "uninstall"}

// ParseStep reads a progress line such as
// "Installing runtime/org.freedesktop.Platform/x86_64/26.08".
func ParseStep(line string) (Step, bool) {
	verb, ref, ok := strings.Cut(strings.TrimSpace(line), " ")
	action, known := verbs[verb]
	if !ok || !known {
		return Step{}, false
	}
	parts := strings.Split(strings.TrimSpace(ref), "/")
	if len(parts) < 2 || (parts[0] != "app" && parts[0] != "runtime") {
		return Step{}, false
	}
	return Step{Action: action, Kind: parts[0], ID: parts[1]}, true
}

// parseSize reads sizes as flatpak prints them, such as "133.4 MB" or
// "912 bytes". Units are decimal, as in GLib's g_format_size.
func parseSize(s string) int64 {
	number, unit, _ := strings.Cut(strings.TrimSpace(strings.ReplaceAll(s, " ", " ")), " ")
	v, err := strconv.ParseFloat(strings.ReplaceAll(number, ",", "."), 64)
	if err != nil {
		return 0
	}
	scale := map[string]float64{"bytes": 1, "byte": 1, "kB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12}[unit]
	if scale == 0 {
		return 0
	}
	return int64(v * scale)
}

func lines(out []byte) []string {
	var found []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			found = append(found, line)
		}
	}
	return found
}

// command builds a flatpak command whose output Nexus can parse: in
// English whatever the desktop's language, and stopped gently on cancel.
func command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "flatpak", args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C.UTF-8", "LANGUAGE=")
	cmd.Stdin = nil
	// SIGTERM lets flatpak end its transaction cleanly; a partial
	// download stays in its cache and resumes next time.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second
	return cmd
}

func output(ctx context.Context, args ...string) ([]byte, error) {
	cmd := command(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, errors.New(strings.TrimPrefix(lastLine(msg), "error: "))
		}
		return nil, err
	}
	return out, nil
}

func stream(ctx context.Context, line func(string), args ...string) error {
	cmd := command(ctx, args...)
	r, w := io.Pipe()
	cmd.Stdout, cmd.Stderr = w, w
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(r)
		scanner.Split(scanLines)
		for scanner.Scan() {
			line(scanner.Text())
		}
		_, _ = io.Copy(io.Discard, r)
	}()
	err := cmd.Wait()
	_ = w.Close()
	<-done
	return err
}

// scanLines splits on \n and on \r, which flatpak uses to redraw a line.
func scanLines(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func lastLine(s string) string {
	parts := strings.Split(s, "\n")
	return strings.TrimSpace(parts[len(parts)-1])
}
