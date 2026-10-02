package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: i18n.T("cmd.version.short"),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "vmctl %s\n", Version)
		},
	}
}
