package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"pandarelax/mestt/internal/transcribe"
)

func newAuthCmd(ctx context.Context, deps dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Configure the OpenAI model and API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(cmd.InOrStdin())

			models := transcribe.Models()
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Select a transcription model:")
			for i, model := range models {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d. %s (%s)\n", i+1, model.Label, model.ID)
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "Model number [1]: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read model selection: %w", err)
			}

			selection := strings.TrimSpace(line)
			selected := models[0]
			if selection != "" {
				var idx int
				if _, err := fmt.Sscanf(selection, "%d", &idx); err != nil || idx < 1 || idx > len(models) {
					return fmt.Errorf("invalid model selection")
				}
				selected = models[idx-1]
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "OpenAI API key: ")
			var apiKey string
			if term.IsTerminal(int(os.Stdin.Fd())) {
				keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				if err != nil {
					return fmt.Errorf("read api key: %w", err)
				}
				apiKey = string(keyBytes)
			} else {
				line, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read api key: %w", err)
				}
				apiKey = strings.TrimSpace(line)
			}

			if err := deps.Auth.SaveOpenAI(ctx, string(selected.ID), apiKey); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved OpenAI configuration with model %s\n", selected.ID)
			return nil
		},
	}
}
