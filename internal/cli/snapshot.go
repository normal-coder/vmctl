package cli

import (
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/output"
)

func newSnapshotCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage virtual machine snapshots",
		Long: "Work with snapshots: list, create, delete and revert.\n" +
			"Reverting requires the VM to be powered off.",
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
		Short: "List snapshots of a virtual machine",
		Args:  cobra.ExactArgs(1),
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
	c.Flags().BoolVar(&tree, "tree", false, "show snapshot hierarchy")
	return c
}

func newSnapshotCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <vm> <name>",
		Short: "Create a snapshot",
		Args:  cobra.ExactArgs(2),
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
		Short: "Delete a snapshot",
		Long: "Delete a snapshot. Child snapshots are re-parented by\n" +
			"default; pass --children to delete them as well.",
		Args: cobra.ExactArgs(2),
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
	c.Flags().BoolVar(&children, "children", false, "also delete child snapshots")
	return c
}

func newSnapshotRevertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revert <vm> <name>",
		Short: "Revert a virtual machine to a snapshot",
		Args:  cobra.ExactArgs(2),
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
