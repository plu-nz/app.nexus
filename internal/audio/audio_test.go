package audio

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseVolume(t *testing.T) {
	cases := map[string]Volume{
		"Volume: 0.39\n":         {Percent: 39},
		"Volume: 1.00 [MUTED]\n": {Percent: 100, Muted: true},
	}
	for in, want := range cases {
		if got, err := parseVolume(in); err != nil || got != want {
			t.Errorf("parseVolume(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	if _, err := parseVolume("nothing"); err == nil {
		t.Error("parseVolume accepted garbage")
	}
}

const dump = `[
 {"id":57,"type":"PipeWire:Interface:Node","info":{"props":{"media.class":"Audio/Sink","node.name":"alsa_output.analog","node.description":"Ryzen HD Audio Analog Stereo"}}},
 {"id":48,"type":"PipeWire:Interface:Node","info":{"props":{"media.class":"Audio/Sink","node.name":"bluez_output.1","node.description":"LG-PL5"}}},
 {"id":60,"type":"PipeWire:Interface:Node","info":{"props":{"media.class":"Audio/Source","node.name":"mic"}}},
 {"id":30,"type":"PipeWire:Interface:Metadata","props":{"metadata.name":"default"},"metadata":[
  {"subject":0,"key":"default.audio.source","value":{"name":"mic"}},
  {"subject":0,"key":"default.audio.sink","value":{"name":"bluez_output.1"}}]}
]`

func TestParseOutputs(t *testing.T) {
	got, err := parseOutputs(dump)
	if err != nil {
		t.Fatal(err)
	}
	want := []Output{{ID: 57, Name: "Ryzen HD Audio Analog Stereo"}, {ID: 48, Name: "LG-PL5", Default: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseOutputs = %+v, want %+v", got, want)
	}
}

func TestMetadataNameAsString(t *testing.T) {
	if got := metadataName([]byte(`"{\"name\":\"hdmi\"}"`)); got != "hdmi" {
		t.Fatalf("metadataName = %q, want hdmi", got)
	}
}

func TestSetOutputOnlyAcceptsListedOutputs(t *testing.T) {
	var calls []string
	m := &Mixer{Run: func(name string, args ...string) (string, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		if name == "pw-dump" {
			return dump, nil
		}
		return "", nil
	}}
	if err := m.SetOutput(99); err == nil {
		t.Fatal("SetOutput accepted an unknown id")
	}
	if err := m.SetOutput(57); err != nil {
		t.Fatal(err)
	}
	if last := calls[len(calls)-1]; last != "wpctl set-default 57" {
		t.Fatalf("SetOutput ran %q", last)
	}
}

func TestSetVolume(t *testing.T) {
	var got string
	m := &Mixer{Run: func(name string, args ...string) (string, error) {
		got = strings.Join(append([]string{name}, args...), " ")
		return "", nil
	}}
	if err := m.SetVolume(45); err != nil || got != "wpctl set-volume @DEFAULT_AUDIO_SINK@ 0.45" {
		t.Fatalf("SetVolume ran %q, %v", got, err)
	}
	if err := m.SetVolume(150); err == nil {
		t.Fatal("SetVolume accepted 150")
	}
	m.Run = func(string, ...string) (string, error) { return "", errors.New("no pipewire") }
	if _, err := m.Volume(); err == nil {
		t.Fatal("Volume hid the error")
	}
}
