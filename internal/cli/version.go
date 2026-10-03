package cli

import (
	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/output"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: i18n.T("cmd.version.short"),
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return output.PrintVersion(cmd.OutOrStdout(), Version, flagJSON)
		},
	}
}
