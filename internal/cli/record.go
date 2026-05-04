package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/popup"
)

func newRecordCmd(ctx context.Context, deps dependencies) *cobra.Command {
	var clipboard bool
	var outputFile string
	var compact bool

	cmd := &cobra.Command{
		Use:   "record",
		Short: "Record audio from the microphone",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shouldLaunchRecordPopup(clipboard, outputFile) {
				launched, err := popup.MaybeLaunchRecorder(ctx)
				if err != nil {
					return err
				}
				if launched {
					return nil
				}
			}

			recordService, cleanup, err := newRecordService()
			if err != nil {
				return err
			}
			defer cleanup()

			target, err := resolveRecordTarget(clipboard, outputFile, compact)
			if err != nil {
				return err
			}

			return recordService.Run(ctx, app.RecordInput{Target: target, Compact: compact})
		},
	}

	cmd.Flags().BoolVarP(&clipboard, "clipboard", "c", false, "Copy transcript to clipboard")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write transcript to a file")
	cmd.Flags().BoolVar(&compact, "compact", false, "Use compact popup-friendly recording UI")
	return cmd
}

func resolveRecordTarget(clipboard bool, outputFile string, compact bool) (output.Target, error) {
	if clipboard && outputFile != "" {
		return output.Target{}, fmt.Errorf("--clipboard and --output cannot be used together")
	}

	target := output.Target{Kind: output.TargetStdout}
	if compact || clipboard {
		target = output.Target{Kind: output.TargetClipboard}
	}
	if outputFile != "" {
		target = output.Target{Kind: output.TargetFile, Path: outputFile}
	}
	return target, nil
}

func shouldLaunchRecordPopup(clipboard bool, outputFile string) bool {
	return clipboard && outputFile == ""
}
