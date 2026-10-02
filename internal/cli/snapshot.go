package cli

import (
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/output"
)

func newSnapshotCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "snapshot",
		Short: i18n.T("cmd.snapshot.short"),
		Long:  i18n.T("cmd.snapshot.long"),
	}
	c.AddCommand(
		newSnapshotListCmd(),
		newSnapshotCreateCmd(),
		newSnapshotDeleteCmd(),
		newSnapshotRevertCmd(),
	)
	return c
}

func newSnapshotListCmd() *cobra.Command {
	var tree bool
	c := &cobra.Command{
		Use:   "list <vm>",
		Short: i18n.T("cmd.snapshot.list.short"),
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			snaps, err := d.Snapshots(cmd.Context(), args[0], tree)
			if err != nil {
				return err
			}
			return output.PrintSnapshots(os.Stdout, args[0], snaps, flagJSON)
		},
	}
	c.Flags().BoolVar(&tree, "tree", false, i18n.T("flag.tree"))
	return c
}

func newSnapshotCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <vm> <name>",
		Short: i18n.T("cmd.snapshot.create.short"),
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := d.SnapshotCreate(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, "snapshot "+args[1], args[0], flagJSON)
		},
	}
}

func newSnapshotDeleteCmd() *cobra.Command {
	var children bool
	c := &cobra.Command{
		Use:   "delete <vm> <name>",
		Short: i18n.T("cmd.snapshot.delete.short"),
		Long:  i18n.T("cmd.snapshot.delete.long"),
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := d.SnapshotDelete(cmd.Context(), args[0], args[1], children); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, "snapshot-delete "+args[1], args[0], flagJSON)
		},
	}
	c.Flags().BoolVar(&children, "children", false, i18n.T("flag.children"))
	return c
}

func newSnapshotRevertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revert <vm> <name>",
		Short: i18n.T("cmd.snapshot.revert.short"),
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := d.SnapshotRevert(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, "snapshot-revert "+args[1], args[0], flagJSON)
		},
	}
}
