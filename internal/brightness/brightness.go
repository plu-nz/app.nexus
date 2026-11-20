// Package brightness reads and sets the display brightness: a backlight
// through brightnessctl, or an external monitor or TV through DDC/CI with
// ddcutil.
package brightness

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// ErrUnsupported means no backlight or DDC/CI display was found.
var ErrUnsupported = errors.New("brightness control is not available")

type kind int

const (
	unknown kind = iota
	backlight
	ddc
)

// Control talks to whichever brightness interface the machine has.
type Control struct {
	// Run executes a command and returns its standard output.
	Run func(name string, args ...string) (string, error)

	mu   sync.Mutex
	kind kind
}

// New returns a Control for the real system.
func New() *Control {
	return &Control{Run: func(name string, args ...string) (string, error) {
		out, err := exec.Command(name, args...).Output()
		return string(out), err
	}}
}

// Get returns the brightness in percent.
func (c *Control) Get() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kind != ddc {
		if out, err := c.Run("brightnessctl", "-m", "-c", "backlight"); err == nil {
			if v, ok := parseBrightnessctl(out); ok {
				c.kind = backlight
				return v, nil
			}
		}
	}
	if out, err := c.Run("ddcutil", "getvcp", "10", "--brief"); err == nil {
		if v, ok := parseDDC(out); ok {
			c.kind = ddc
			return v, nil
		}
	}
	c.kind = unknown
	return 0, ErrUnsupported
}

// Set changes the brightness, in percent from 1 to 100.
func (c *Control) Set(percent int) error {
	if percent < 1 || percent > 100 {
		return fmt.Errorf("brightness must be between 1 and 100, not %d", percent)
	}
	c.mu.Lock()
	k := c.kind
	c.mu.Unlock()
	if k == unknown {
		if _, err := c.Get(); err != nil {
			return err
		}
		c.mu.Lock()
		k = c.kind
		c.mu.Unlock()
	}
	var err error
	if k == ddc {
		_, err = c.Run("ddcutil", "setvcp", "10", strconv.Itoa(percent))
	} else {
		_, err = c.Run("brightnessctl", "-q", "-c", "backlight", "set", strconv.Itoa(percent)+"%")
	}
	return err
}

// parseBrightnessctl reads the first device of `brightnessctl -m`:
// "name,class,current,percent%,max".
func parseBrightnessctl(out string) (int, bool) {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	fields := strings.Split(line, ",")
	if len(fields) < 5 {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSuffix(fields[3], "%"))
	return v, err == nil
}

// parseDDC reads `ddcutil getvcp 10 --brief`: "VCP 10 C <current> <max>".
func parseDDC(out string) (int, bool) {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 5 || f[0] != "VCP" || f[1] != "10" || f[2] != "C" {
			continue
		}
		cur, err1 := strconv.Atoi(f[3])
		max, err2 := strconv.Atoi(f[4])
		if err1 != nil || err2 != nil || max <= 0 {
			return 0, false
		}
		return (cur*100 + max/2) / max, true
	}
	return 0, false
}
