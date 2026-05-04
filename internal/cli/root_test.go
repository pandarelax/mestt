package cli

import (
	"reflect"
	"testing"
)

func TestNormalizeRootArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "defaults to record", args: nil, want: []string{"record"}},
		{name: "leading flag becomes record", args: []string{"-c"}, want: []string{"record", "-c"}},
		{name: "leading long flag becomes record", args: []string{"--compact", "-c"}, want: []string{"record", "--compact", "-c"}},
		{name: "explicit subcommand unchanged", args: []string{"record", "-c"}, want: []string{"record", "-c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRootArgs(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeRootArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
