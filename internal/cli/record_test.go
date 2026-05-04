package cli

import (
	"testing"

	"pandarelax/mestt/internal/output"
)

func TestResolveRecordTarget(t *testing.T) {
	tests := []struct {
		name       string
		clipboard  bool
		outputFile string
		compact    bool
		want       output.Target
		wantErr    bool
	}{
		{name: "default stdout", want: output.Target{Kind: output.TargetStdout}},
		{name: "clipboard flag", clipboard: true, want: output.Target{Kind: output.TargetClipboard}},
		{name: "compact defaults to clipboard", compact: true, want: output.Target{Kind: output.TargetClipboard}},
		{name: "output file wins", compact: true, outputFile: "out.txt", want: output.Target{Kind: output.TargetFile, Path: "out.txt"}},
		{name: "clipboard and output conflict", clipboard: true, outputFile: "out.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveRecordTarget(tt.clipboard, tt.outputFile, tt.compact)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveRecordTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("resolveRecordTarget() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestShouldLaunchRecordPopup(t *testing.T) {
	tests := []struct {
		name       string
		clipboard  bool
		outputFile string
		want       bool
	}{
		{name: "clipboard recording launches popup", clipboard: true, want: true},
		{name: "output file disables popup", clipboard: true, outputFile: "out.txt", want: false},
		{name: "stdout recording does not launch popup", clipboard: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLaunchRecordPopup(tt.clipboard, tt.outputFile); got != tt.want {
				t.Fatalf("shouldLaunchRecordPopup() = %v, want %v", got, tt.want)
			}
		})
	}
}
