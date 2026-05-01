package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newHistoryCmd(ctx context.Context, deps dependencies) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List previous transcriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := deps.History.List(ctx, limit)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				preview := strings.ReplaceAll(entry.Transcript, "\n", " ")
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\n", entry.ID, entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.ModelID, preview)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of history rows to show")
	return cmd
}
