// Package audio controls the default PipeWire output: its volume, mute,
// and which device it is. It uses wpctl and pw-dump from PipeWire and
// WirePlumber.
package audio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const sink = "@DEFAULT_AUDIO_SINK@"

// Volume is the default output's level.
type Volume struct {
	Percent int  `json:"percent"`
	Muted   bool `json:"muted"`
}

// Output is an audio device the sound can go to.
type Output struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

// Mixer runs the PipeWire tools.
type Mixer struct {
	// Run executes a command and returns its standard output.
	Run func(name string, args ...string) (string, error)
}

// New returns a Mixer for the real system.
func New() *Mixer {
	return &Mixer{Run: func(name string, args ...string) (string, error) {
		out, err := exec.Command(name, args...).Output()
		return string(out), err
	}}
}

// Volume returns the default output's volume.
func (m *Mixer) Volume() (Volume, error) {
	out, err := m.Run("wpctl", "get-volume", sink)
	if err != nil {
		return Volume{}, err
	}
	return parseVolume(out)
}

// SetVolume sets the default output's volume, from 0 to 100 percent.
func (m *Mixer) SetVolume(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("volume must be between 0 and 100, not %d", percent)
	}
	_, err := m.Run("wpctl", "set-volume", sink, fmt.Sprintf("%.2f", float64(percent)/100))
	return err
}

// ToggleMute mutes or unmutes the default output.
func (m *Mixer) ToggleMute() error {
	_, err := m.Run("wpctl", "set-mute", sink, "toggle")
	return err
}

// Outputs lists the audio outputs, marking the default one.
func (m *Mixer) Outputs() ([]Output, error) {
	out, err := m.Run("pw-dump")
	if err != nil {
		return nil, err
	}
	return parseOutputs(out)
}

// SetOutput makes an output the default. Only listed outputs are accepted.
func (m *Mixer) SetOutput(id int) error {
	outputs, err := m.Outputs()
	if err != nil {
		return err
	}
	for _, o := range outputs {
		if o.ID == id {
			_, err := m.Run("wpctl", "set-default", strconv.Itoa(id))
			return err
		}
	}
	return errors.New("unknown audio output")
}

// parseVolume reads `wpctl get-volume`: "Volume: 0.39" or "Volume: 0.39 [MUTED]".
func parseVolume(out string) (Volume, error) {
	f := strings.Fields(out)
	if len(f) < 2 || f[0] != "Volume:" {
		return Volume{}, fmt.Errorf("unexpected wpctl output %q", strings.TrimSpace(out))
	}
	v, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return Volume{}, fmt.Errorf("unexpected wpctl output %q", strings.TrimSpace(out))
	}
	return Volume{Percent: int(v*100 + 0.5), Muted: strings.Contains(out, "[MUTED]")}, nil
}

type pwObject struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Props struct {
		MetadataName string `json:"metadata.name"`
	} `json:"props"`
	Info struct {
		Props struct {
			MediaClass  string `json:"media.class"`
			Name        string `json:"node.name"`
			Description string `json:"node.description"`
			Nick        string `json:"node.nick"`
		} `json:"props"`
	} `json:"info"`
	Metadata []struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	} `json:"metadata"`
}

// parseOutputs reads pw-dump's JSON: the Audio/Sink nodes and the
// "default.audio.sink" entry of the default metadata.
func parseOutputs(dump string) ([]Output, error) {
	var objects []pwObject
	if err := json.Unmarshal([]byte(dump), &objects); err != nil {
		return nil, fmt.Errorf("reading pw-dump: %w", err)
	}
	defaultName := ""
	for _, o := range objects {
		if o.Type != "PipeWire:Interface:Metadata" || o.Props.MetadataName != "default" {
			continue
		}
		for _, m := range o.Metadata {
			if m.Key == "default.audio.sink" {
				defaultName = metadataName(m.Value)
			}
		}
	}
	outputs := []Output{}
	for _, o := range objects {
		p := o.Info.Props
		if o.Type != "PipeWire:Interface:Node" || p.MediaClass != "Audio/Sink" {
			continue
		}
		name := p.Description
		if name == "" {
			name = p.Nick
		}
		if name == "" {
			name = p.Name
		}
		outputs = append(outputs, Output{ID: o.ID, Name: name, Default: p.Name != "" && p.Name == defaultName})
	}
	return outputs, nil
}

// metadataName reads {"name": "..."}, which pw-dump writes as an object or,
// in older versions, as a JSON string.
func metadataName(raw json.RawMessage) string {
	var v struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &v) == nil && v.Name != "" {
		return v.Name
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && json.Unmarshal([]byte(s), &v) == nil {
		return v.Name
	}
	return ""
}
