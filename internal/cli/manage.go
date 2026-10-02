package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
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
		Short: i18n.T("cmd.clone.short"),
		Long:  i18n.T("cmd.clone.long"),
		Args:  cobra.ExactArgs(1),
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
	c.Flags().StringVarP(&name, "name", "n", "", i18n.T("flag.clone.name"))
	c.Flags().BoolVar(&full, "full", false, i18n.T("flag.full"))
	c.Flags().StringVar(&snapshot, "snapshot", "", i18n.T("flag.snapshot"))
	c.Flags().StringVar(&path, "path", "", i18n.T("flag.path"))
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
		Short: i18n.T("cmd.create.short"),
		Long:  i18n.T("cmd.create.long"),
		Args:  cobra.ExactArgs(1),
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
	c.Flags().StringVarP(&from, "from", "f", "", i18n.T("flag.from"))
	c.Flags().BoolVar(&full, "full", false, "create an independent full copy instead of a linked clone")
	c.Flags().StringVar(&snapshot, "snapshot", "", "base snapshot name for the clone")
	c.Flags().StringVar(&path, "path", "", "destination bundle directory or .vmx path")
	c.Flags().IntVar(&memory, "memory", 0, i18n.T("flag.memory"))
	c.Flags().IntVar(&cpus, "cpus", 0, i18n.T("flag.cpus"))
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
		Short: i18n.T("cmd.set.short"),
		Long:  i18n.T("cmd.set.long"),
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
				return fmt.Errorf("%s", i18n.T("err.set.flags"))
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
	c.Flags().StringVar(&name, "name", "", i18n.T("flag.set.name"))
	c.Flags().IntVar(&memory, "memory", 0, i18n.T("flag.set.memory"))
	c.Flags().IntVar(&cpus, "cpus", 0, i18n.T("flag.set.cpus"))
	return c
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <vm>",
		Short: i18n.T("cmd.delete.short"),
		Long:  i18n.T("cmd.delete.long"),
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
