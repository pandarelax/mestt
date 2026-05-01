package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/output"
)

func newTranscribeCmd(ctx context.Context, deps dependencies) *cobra.Command {
	var clipboard bool
	var outputFile string

	cmd := &cobra.Command{
		Use:   "transcribe <file>",
		Short: "Transcribe an existing audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if clipboard && outputFile != "" {
				return fmt.Errorf("--clipboard and --output cannot be used together")
			}

			audioPath := args[0]
			if err := ensureFileExists(audioPath); err != nil {
				return err
			}

			target := output.Target{Kind: output.TargetStdout}
			if clipboard {
				target = output.Target{Kind: output.TargetClipboard}
			}
			if outputFile != "" {
				target = output.Target{Kind: output.TargetFile, Path: outputFile}
			}

			_, err := deps.Transcribe.Run(ctx, app.TranscribeInput{AudioPath: audioPath, Target: target})
			return err
		},
	}

	cmd.Flags().BoolVarP(&clipboard, "clipboard", "c", false, "Copy transcript to clipboard")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write transcript to a file")
	return cmd
}
