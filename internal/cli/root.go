package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/history"
	"pandarelax/mestt/internal/logging"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/paths"
	"pandarelax/mestt/internal/secret"
	"pandarelax/mestt/internal/transcribe"
	recordtui "pandarelax/mestt/internal/tui/record"
)

const version = "0.1.0-dev"

type dependencies struct {
	Paths paths.Paths
}

func Run(ctx context.Context, args []string) error {
	deps := dependencies{
		Paths: paths.Resolve(),
	}

	cmd := newRootCmd(ctx, deps)
	args = normalizeRootArgs(args)
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

func newAuthService() app.AuthService {
	return app.AuthService{Secrets: secret.NewFileStore()}
}

func newAudioRecorder() audio.Recorder {
	return audio.NewRecorder()
}

func newTranscribeService() (app.TranscribeService, func(), error) {
	if _, err := logging.Setup(); err != nil {
		return app.TranscribeService{}, nil, err
	}
	historyStore, err := history.Open()
	if err != nil {
		return app.TranscribeService{}, nil, err
	}
	service := app.TranscribeService{
		Secrets: secret.NewFileStore(),
		History: historyStore,
		Output:  output.Writer{},
		Client:  transcribe.OpenAIClient{},
		Local:   transcribe.LocalClient{},
	}
	return service, func() { _ = historyStore.Close() }, nil
}

func newRecordService() (app.RecordService, func(), error) {
	transcribeService, cleanup, err := newTranscribeService()
	if err != nil {
		return app.RecordService{}, nil, err
	}
	service := app.RecordService{
		Recorder:   newAudioRecorder(),
		RunUI:      recordtui.Runner{},
		Transcribe: transcribeService,
		Prepare:    transcribeService,
	}
	return service, cleanup, nil
}

func newHistoryService() (app.HistoryService, func(), error) {
	if _, err := logging.Setup(); err != nil {
		return app.HistoryService{}, nil, err
	}
	historyStore, err := history.Open()
	if err != nil {
		return app.HistoryService{}, nil, err
	}
	return app.HistoryService{Store: historyStore}, func() { _ = historyStore.Close() }, nil
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

func normalizeRootArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"record"}
	}
	if strings.HasPrefix(args[0], "-") {
		return append([]string{"record"}, args...)
	}
	return args
}
