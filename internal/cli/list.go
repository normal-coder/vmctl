package cli

import (
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/output"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List virtual machines",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			vms, err := d.List(cmd.Context())
			if err != nil {
				return err
			}
			return output.PrintVMs(os.Stdout, vms, flagJSON)
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <vm>",
		Short: "Show detailed information about a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			info, err := d.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.PrintInfo(os.Stdout, info, flagJSON)
		},
	}
}
