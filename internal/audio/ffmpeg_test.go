package audio

import (
	"reflect"
	"testing"
)

func TestBuildRecordArgsLinux(t *testing.T) {
	got, err := buildRecordArgs("linux", RecordOptions{Device: "default", SampleRate: 16000}, "/tmp/out.wav")
	if err != nil {
		t.Fatalf("buildRecordArgs() error = %v", err)
	}
	want := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "pulse", "-i", "default", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", "/tmp/out.wav"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestBuildRecordArgsDarwin(t *testing.T) {
	got, err := buildRecordArgs("darwin", RecordOptions{Device: "1", SampleRate: 44100}, "/tmp/out.wav")
	if err != nil {
		t.Fatalf("buildRecordArgs() error = %v", err)
	}
	want := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "avfoundation", "-i", ":1", "-ar", "44100", "-ac", "1", "-c:a", "pcm_s16le", "/tmp/out.wav"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestParseDarwinDevices(t *testing.T) {
	output := `[AVFoundation indev @ 0x1] AVFoundation video devices:
[AVFoundation indev @ 0x1] [0] FaceTime HD Camera
[AVFoundation indev @ 0x1] AVFoundation audio devices:
[AVFoundation indev @ 0x1] [0] MacBook Pro Microphone
[AVFoundation indev @ 0x1] [1] USB Audio Device`
	devices := parseDevices("darwin", output)
	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}
	if devices[0].Name != "MacBook Pro Microphone" || !devices[0].Default {
		t.Fatalf("unexpected first device: %#v", devices[0])
	}
}

func TestParseLinuxDevices(t *testing.T) {
	output := `Auto-detected sources for pulse:
  * default [Default Input Device]
    alsa_input.usb-0 [USB Microphone]`
	devices := parseDevices("linux", output)
	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}
	if devices[0].ID != "default" || !devices[0].Default {
		t.Fatalf("unexpected first device: %#v", devices[0])
	}
}
