package popup

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestMaybeLaunchRecorderStartsChildCommand(t *testing.T) {
	var gotName string
	var gotArgs []string
	var gotEnv []string

	l := launcher{
		lookPath: func(name string) (string, error) {
			if name == "ghostty" {
				return "/bin/ghostty", nil
			}
			return "", errors.New("missing")
		},
		executable: func() (string, error) { return "/tmp/mestt", nil },
		start: func(_ context.Context, name string, args []string, env []string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
		environ: func() []string { return []string{"PATH=/bin"} },
	}

	launched, err := l.maybeLaunchRecorder(context.Background())
	if err != nil {
		t.Fatalf("maybeLaunchRecorder() error = %v", err)
	}
	if !launched {
		t.Fatalf("maybeLaunchRecorder() launched = false, want true")
	}
	if gotName != "/bin/ghostty" {
		t.Fatalf("name = %q, want %q", gotName, "/bin/ghostty")
	}
	wantArgs := []string{"-e", "/tmp/mestt", "record", "--compact", "-c"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !contains(gotEnv, childEnvVar+"=1") {
		t.Fatalf("env = %#v, want %q", gotEnv, childEnvVar+"=1")
	}
}

func TestMaybeLaunchRecorderFallsBackWhenNoTerminalFound(t *testing.T) {
	l := launcher{
		lookPath:   func(string) (string, error) { return "", errors.New("missing") },
		executable: func() (string, error) { return "/tmp/mestt", nil },
		start:      func(context.Context, string, []string, []string) error { return nil },
		environ:    func() []string { return nil },
	}

	launched, err := l.maybeLaunchRecorder(context.Background())
	if err != nil {
		t.Fatalf("maybeLaunchRecorder() error = %v", err)
	}
	if launched {
		t.Fatalf("maybeLaunchRecorder() launched = true, want false")
	}
}

func TestMaybeLaunchRecorderReturnsLastLaunchError(t *testing.T) {
	l := launcher{
		lookPath: func(name string) (string, error) {
			if name == "ghostty" || name == "kitty" {
				return "/bin/" + name, nil
			}
			return "", errors.New("missing")
		},
		executable: func() (string, error) { return "/tmp/mestt", nil },
		start: func(context.Context, string, []string, []string) error {
			return errors.New("boom")
		},
		environ: func() []string { return nil },
	}

	launched, err := l.maybeLaunchRecorder(context.Background())
	if err == nil {
		t.Fatalf("maybeLaunchRecorder() error = nil, want non-nil")
	}
	if launched {
		t.Fatalf("maybeLaunchRecorder() launched = true, want false")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
