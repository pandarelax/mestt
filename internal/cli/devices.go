package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newListDevicesCmd(ctx context.Context, deps dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list-devices",
		Short: "List available audio input devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			devices, err := deps.Audio.ListDevices(ctx)
			if err != nil {
				return err
			}
			for _, device := range devices {
				defaultLabel := ""
				if device.Default {
					defaultLabel = " [default]"
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s%s\n", device.ID, device.Name, defaultLabel)
			}
			return nil
		},
	}
}
