package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/daemon"
)

const version = "0.1.0-dev"

func main() {
	if err := newRootCmd(context.Background()).Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mesttd",
		Short:        "mestt trigger helper",
		SilenceUsage: true,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Show current trigger configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trigger_command=%s\n", cfg.Daemon.TriggerCommand)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trigger_args=%v\n", cfg.Daemon.TriggerArgs)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "trigger",
		Short: "Launch the configured popup trigger command",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return daemon.Trigger(ctx, daemon.TriggerOptions{
				Command: cfg.Daemon.TriggerCommand,
				Args:    cfg.Daemon.TriggerArgs,
			})
		},
	})

	cmd.SetArgs(os.Args[1:])
	return cmd
}
