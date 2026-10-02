package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
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
		Args:  exactArgs(1),
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
		c.Flags().BoolVar(&f.hard, "hard", false, i18n.T("flag.hard"))
		if p.use == "start" {
			c.Flags().BoolVar(&f.noGUI, "nogui", false, i18n.T("flag.nogui"))
		}
	}
	return c
}

func newStartCmd() *cobra.Command {
	return powerCommand{
		use:       "start",
		short:     i18n.T("cmd.start.short"),
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Start(ctx, ref, driver.StartOptions{NoGUI: f.noGUI})
		},
	}.cmd()
}

func newStopCmd() *cobra.Command {
	return powerCommand{
		use:       "stop",
		short:     i18n.T("cmd.stop.short"),
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Stop(ctx, ref, driver.StopOptions{Hard: f.hard})
		},
	}.cmd()
}

func newSuspendCmd() *cobra.Command {
	return powerCommand{
		use:       "suspend",
		short:     i18n.T("cmd.suspend.short"),
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Suspend(ctx, ref, driver.SuspendOptions{Hard: f.hard})
		},
	}.cmd()
}

func newResumeCmd() *cobra.Command {
	return powerCommand{
		use:   "resume",
		short: i18n.T("cmd.resume.short"),
		run: func(ctx context.Context, d driver.Driver, ref string, _ *powerFlags) error {
			return d.Resume(ctx, ref)
		},
	}.cmd()
}

func newPauseCmd() *cobra.Command {
	return powerCommand{
		use:   "pause",
		short: i18n.T("cmd.pause.short"),
		run: func(ctx context.Context, d driver.Driver, ref string, _ *powerFlags) error {
			return d.Pause(ctx, ref)
		},
	}.cmd()
}

func newResetCmd() *cobra.Command {
	return powerCommand{
		use:       "reset",
		short:     i18n.T("cmd.reset.short"),
		withFlags: true,
		run: func(ctx context.Context, d driver.Driver, ref string, f *powerFlags) error {
			return d.Reset(ctx, ref, driver.ResetOptions{Hard: f.hard})
		},
	}.cmd()
}
