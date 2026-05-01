package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/output"
)

func newRecordCmd(ctx context.Context, deps dependencies) *cobra.Command {
	var clipboard bool
	var outputFile string

	cmd := &cobra.Command{
		Use:   "record",
		Short: "Record audio from the microphone",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clipboard && outputFile != "" {
				return fmt.Errorf("--clipboard and --output cannot be used together")
			}

			target := output.Target{Kind: output.TargetStdout}
			if clipboard {
				target = output.Target{Kind: output.TargetClipboard}
			}
			if outputFile != "" {
				target = output.Target{Kind: output.TargetFile, Path: outputFile}
			}

			return deps.Record.Run(ctx, app.RecordInput{Target: target})
		},
	}

	cmd.Flags().BoolVarP(&clipboard, "clipboard", "c", false, "Copy transcript to clipboard")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write transcript to a file")
	return cmd
}
