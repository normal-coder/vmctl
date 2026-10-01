package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/output"
)

type powerFlags struct {
	hard  bool
	noGUI bool
}

// powerCommand wires one power operation into a cobra command.
type powerCommand struct {
	use   string
	short string
	// withFlags adds --hard (and --nogui for start).
	withFlags bool
	run       func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error
}

func (p powerCommand) cmd() *cobra.Command {
	var f powerFlags
	c := &cobra.Command{
		Use:   p.use,
		Short: p.short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if err := p.run(cmd.Context(), d, args[0], &f); err != nil {
				return err
			}
			return output.PrintOK(os.Stdout, p.use, args[0], flagJSON)
		},
	}
	if p.withFlags {
		c.Flags().BoolVar(&f.hard, "hard", false, "force the operation (skip guest shutdown)")
		if p.use == "start" {
			c.Flags().BoolVar(&f.noGUI, "nogui", false, "start without opening a window")
		}
	}
	return c
}

func newStartCmd() *cobra.Command {
	return powerCommand{
		use:       "start",
		short:     "Start or resume a virtual machine",
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Start(ctx, ref, driver.StartOptions{NoGUI: f.noGUI})
		},
	}.cmd()
}

func newStopCmd() *cobra.Command {
	return powerCommand{
		use:       "stop",
		short:     "Power off a virtual machine",
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Stop(ctx, ref, driver.StopOptions{Hard: f.hard})
		},
	}.cmd()
}

func newSuspendCmd() *cobra.Command {
	return powerCommand{
		use:       "suspend",
		short:     "Suspend a virtual machine to disk",
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Suspend(ctx, ref, driver.SuspendOptions{Hard: f.hard})
		},
	}.cmd()
}

func newResumeCmd() *cobra.Command {
	return powerCommand{
		use:   "resume",
		short: "Resume a suspended or paused virtual machine",
		run: func(ctx context.Context, d driver.Driver, ref string, _ *powerFlags) error {
			return d.Resume(ctx, ref)
		},
	}.cmd()
}

func newPauseCmd() *cobra.Command {
	return powerCommand{
		use:   "pause",
		short: "Pause a running virtual machine",
		run: func(ctx context.Context, d driver.Driver, ref string, _ *powerFlags) error {
			return d.Pause(ctx, ref)
		},
	}.cmd()
}

func newResetCmd() *cobra.Command {
	return powerCommand{
		use:       "reset",
		short:     "Reset a virtual machine",
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Reset(ctx, ref, driver.ResetOptions{Hard: f.hard})
		},
	}.cmd()
}
