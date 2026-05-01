package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/history"
	"pandarelax/mestt/internal/logging"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/paths"
	"pandarelax/mestt/internal/secret"
	"pandarelax/mestt/internal/transcribe"
)

const version = "0.1.0-dev"

type dependencies struct {
	Auth       app.AuthService
	Audio      audio.Recorder
	Transcribe app.TranscribeService
	Record     app.RecordService
	History    app.HistoryService
	Paths      paths.Paths
}

func Run(ctx context.Context, args []string) error {
	if _, err := logging.Setup(); err != nil {
		return err
	}

	historyStore, err := history.Open()
	if err != nil {
		return err
	}
	defer historyStore.Close()

	audioRecorder := audio.NewRecorder()

	deps := dependencies{
		Auth:  app.AuthService{Secrets: secret.NewFileStore()},
		Audio: audioRecorder,
		Transcribe: app.TranscribeService{
			Secrets: secret.NewFileStore(),
			History: historyStore,
			Output:  output.Writer{},
			Client:  transcribe.OpenAIClient{},
		},
		Record: app.RecordService{
			Recorder: audioRecorder,
			Transcribe: app.TranscribeService{
				Secrets: secret.NewFileStore(),
				History: historyStore,
				Output:  output.Writer{},
				Client:  transcribe.OpenAIClient{},
			},
		},
		History: app.HistoryService{Store: historyStore},
		Paths:   paths.Resolve(),
	}

	cmd := newRootCmd(ctx, deps)
	if len(args) == 0 {
		args = []string{"record"}
	}
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCmd(ctx context.Context, deps dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mestt",
		Short:        "Local-first speech-to-text CLI",
		SilenceUsage: true,
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newConfigCmd(deps))
	cmd.AddCommand(newAuthCmd(ctx, deps))
	cmd.AddCommand(newTranscribeCmd(ctx, deps))
	cmd.AddCommand(newHistoryCmd(ctx, deps))
	cmd.AddCommand(newRecordCmd(ctx, deps))
	cmd.AddCommand(newListDevicesCmd(ctx, deps))

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

func newConfigCmd(deps dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print the config file path",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), deps.Paths.ConfigFile)
		},
	}
}

func ensureFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}
