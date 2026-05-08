package daemon

import (
	"context"
	"errors"
	"testing"
)

func TestTriggerRequiresCommand(t *testing.T) {
	err := launcher{}.Trigger(context.Background(), TriggerOptions{})
	if err == nil {
		t.Fatal("Trigger() error = nil, want error")
	}
}

func TestTriggerReportsMissingCommand(t *testing.T) {
	err := launcher{lookPath: func(name string) (string, error) {
		return "", errors.New("missing")
	}}.Trigger(context.Background(), TriggerOptions{Command: "mestt-gui"})
	if err == nil {
		t.Fatal("Trigger() error = nil, want error")
	}
}

func TestTriggerStartsResolvedCommand(t *testing.T) {
	var (
		gotName string
		gotArgs []string
	)
	err := launcher{
		lookPath: func(name string) (string, error) {
			return "/tmp/mestt-gui", nil
		},
		start: func(_ context.Context, name string, args []string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}.Trigger(context.Background(), TriggerOptions{Command: "mestt-gui", Args: []string{"--flag"}})
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if gotName != "/tmp/mestt-gui" {
		t.Fatalf("started name = %q, want %q", gotName, "/tmp/mestt-gui")
	}
	if len(gotArgs) != 1 || gotArgs[0] != "--flag" {
		t.Fatalf("started args = %#v, want [--flag]", gotArgs)
	}
}
