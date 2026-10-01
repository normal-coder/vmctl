package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/output"
)

func newCloneCmd() *cobra.Command {
	var (
		name     string
		full     bool
		snapshot string
		path     string
	)
	c := &cobra.Command{
		Use:   "clone <vm>",
		Short: "Clone a virtual machine",
		Long: "Clone a virtual machine. By default a linked clone is created\n" +
			"(fast; keeps a base snapshot on the source). Use --full for an\n" +
			"independent copy. The source must be powered off.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			dest, err := d.Clone(cmd.Context(), args[0], driver.CloneOptions{
				Name:     name,
				Full:     full,
				Snapshot: snapshot,
				Path:     path,
			})
			if err != nil {
				return err
			}
			return output.PrintCreated(os.Stdout, "clone", args[0], dest, flagJSON)
		},
	}
	c.Flags().StringVarP(&name, "name", "n", "", "name for the new virtual machine (required)")
	c.Flags().BoolVar(&full, "full", false, "create an independent full copy instead of a linked clone")
	c.Flags().StringVar(&snapshot, "snapshot", "", "base snapshot name for the clone")
	c.Flags().StringVar(&path, "path", "", "destination bundle directory or .vmx path")
	_ = c.MarkFlagRequired("name")
	return c
}

func newCreateCmd() *cobra.Command {
	var (
		from     string
		full     bool
		snapshot string
		path     string
		memory   int
		cpus     int
	)
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a virtual machine from an existing one",
		Long: "Provision a new virtual machine by cloning a source VM.\n" +
			"Creating a bare VM from scratch is not supported by the vmrun\n" +
			"backend (Fusion provides no createVM command).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			dest, err := d.Create(cmd.Context(), driver.CreateOptions{
				Name:     args[0],
				From:     from,
				Full:     full,
				Snapshot: snapshot,
				Path:     path,
				MemoryMB: memory,
				CPUs:     cpus,
			})
			if err != nil {
				return err
			}
			return output.PrintCreated(os.Stdout, "create", args[0], dest, flagJSON)
		},
	}
	c.Flags().StringVarP(&from, "from", "f", "", "source virtual machine to clone from (required)")
	c.Flags().BoolVar(&full, "full", false, "create an independent full copy instead of a linked clone")
	c.Flags().StringVar(&snapshot, "snapshot", "", "base snapshot name for the clone")
	c.Flags().StringVar(&path, "path", "", "destination bundle directory or .vmx path")
	c.Flags().IntVar(&memory, "memory", 0, "memory in MB for the new VM (0 = keep source)")
	c.Flags().IntVar(&cpus, "cpus", 0, "number of CPUs for the new VM (0 = keep source)")
	_ = c.MarkFlagRequired("from")
	return c
}

func newSetCmd() *cobra.Command {
	var (
		name   string
		memory int
		cpus   int
	)
	c := &cobra.Command{
		Use:   "set <vm>",
		Short: "Change virtual machine configuration",
		Long:  "Change memory, CPU count or display name of a powered-off VM.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := driver.SetOptions{}
			if cmd.Flags().Changed("name") {
				opts.Name = &name
			}
			if cmd.Flags().Changed("memory") {
				opts.MemoryMB = &memory
			}
			if cmd.Flags().Changed("cpus") {
				opts.CPUs = &cpus
			}
			if opts.Empty() {
				return fmt.Errorf("specify at least one of --name, --memory, --cpus")
			}
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := d.Set(cmd.Context(), args[0], opts); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, "set", args[0], flagJSON)
		},
	}
	c.Flags().StringVar(&name, "name", "", "new display name")
	c.Flags().IntVar(&memory, "memory", 0, "memory in MB")
	c.Flags().IntVar(&cpus, "cpus", 0, "number of CPUs")
	return c
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <vm>",
		Short: "Permanently delete a virtual machine and its files",
		Long:  "Permanently delete a virtual machine, including all files on disk. This cannot be undone.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := d.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, "delete", args[0], flagJSON)
		},
	}
}
