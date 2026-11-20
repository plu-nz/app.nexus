package brightness

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fake struct {
	out   map[string]string
	calls []string
}

func (f *fake) run(name string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, cmd)
	for prefix, out := range f.out {
		if strings.HasPrefix(cmd, prefix) {
			return out, nil
		}
	}
	return "", errors.New("exit status 1")
}

func TestBacklight(t *testing.T) {
	f := &fake{out: map[string]string{"brightnessctl -m": "amdgpu_bl1,backlight,55049,84%,65535\n", "brightnessctl -q": ""}}
	c := &Control{Run: f.run}
	if v, err := c.Get(); err != nil || v != 84 {
		t.Fatalf("Get = %d, %v; want 84", v, err)
	}
	if err := c.Set(60); err != nil {
		t.Fatal(err)
	}
	if last := f.calls[len(f.calls)-1]; last != "brightnessctl -q -c backlight set 60%" {
		t.Fatalf("Set ran %q", last)
	}
}

func TestDDCWhenThereIsNoBacklight(t *testing.T) {
	f := &fake{out: map[string]string{"brightnessctl -m": "", "ddcutil getvcp": "VCP 10 C 30 60\n", "ddcutil setvcp": ""}}
	c := &Control{Run: f.run}
	if v, err := c.Get(); err != nil || v != 50 {
		t.Fatalf("Get = %d, %v; want 50", v, err)
	}
	f.calls = nil
	if err := c.Set(70); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.calls, []string{"ddcutil setvcp 10 70"}) {
		t.Fatalf("Set ran %v", f.calls)
	}
}

func TestUnsupported(t *testing.T) {
	c := &Control{Run: (&fake{}).run}
	if _, err := c.Get(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Get error = %v, want ErrUnsupported", err)
	}
	if err := c.Set(50); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Set error = %v, want ErrUnsupported", err)
	}
	if err := c.Set(0); err == nil {
		t.Fatal("Set(0) was accepted")
	}
}
