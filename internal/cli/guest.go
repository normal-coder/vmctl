package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/output"
)

func newExecCmd() *cobra.Command {
	var guestUser, password string
	c := &cobra.Command{
		Use:   "exec <vm> -- <command>",
		Short: i18n.T("cmd.exec.short"),
		Long:  i18n.T("cmd.exec.long"),
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			if password == "" {
				password = os.Getenv("VMCTL_GUEST_PASSWORD")
			}
			script := strings.Join(args[1:], " ")
			out, code, err := d.Exec(cmd.Context(), args[0], driver.ExecOptions{
				Script:   script,
				User:     guestUser,
				Password: password,
			})
			if err != nil {
				return err
			}
			if flagJSON {
				return output.PrintExec(os.Stdout, args[0], script, out, code)
			}
			fmt.Print(out)
			if code != 0 {
				if code < 0 {
					code = 1
				}
				os.Exit(code)
			}
			return nil
		},
	}
	c.Flags().StringVarP(&guestUser, "user", "u", "", i18n.T("flag.exec.user"))
	c.Flags().StringVarP(&password, "password", "p", "", i18n.T("flag.exec.password"))
	return c
}

func newShellCmd() *cobra.Command {
	var (
		login    string
		port     string
		identity string
		wait     bool
		dryRun   bool
	)
	c := &cobra.Command{
		Use:   "shell <vm>",
		Short: i18n.T("cmd.shell.short"),
		Long:  i18n.T("cmd.shell.long"),
		// Exactly one <vm> before --; anything after -- is for ssh.
		Args: func(cmd *cobra.Command, args []string) error {
			dash := cmd.Flags().ArgsLenAtDash()
			if dash < 0 {
				dash = len(args)
			}
			if dash != 1 {
				return fmt.Errorf(i18n.T("err.shell.args"), dash)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			ip, err := d.GuestIP(cmd.Context(), args[0], wait)
			if err != nil {
				return err
			}
			if login == "" {
				if u, err := user.Current(); err == nil {
					login = u.Username
				}
			}
			// Everything after -- belongs to ssh.
			var extra []string
			if dash := cmd.Flags().ArgsLenAtDash(); dash >= 0 && dash < len(args) {
				extra = args[dash:]
				args = args[:dash]
			}
			sshArgs := buildSSHArgs(login, ip, port, identity, extra)
			if dryRun {
				fmt.Println(joinCommand("ssh", sshArgs))
				return nil
			}
			return runSSH("ssh", sshArgs)
		},
	}
	c.Flags().StringVarP(&login, "user", "u", "", i18n.T("flag.shell.user"))
	c.Flags().StringVarP(&port, "port", "", "22", i18n.T("flag.shell.port"))
	c.Flags().StringVarP(&identity, "identity", "i", "", i18n.T("flag.shell.identity"))
	c.Flags().BoolVar(&wait, "wait", false, i18n.T("flag.wait"))
	c.Flags().BoolVar(&dryRun, "dry-run", false, i18n.T("flag.dry-run"))
	return c
}

func newIPCmd() *cobra.Command {
	var wait bool
	c := &cobra.Command{
		Use:   "ip <vm>",
		Short: i18n.T("cmd.ip.short"),
		Long:  i18n.T("cmd.ip.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDriver()
			if err != nil {
				return err
			}
			ip, err := d.GuestIP(cmd.Context(), args[0], wait)
			if err != nil {
				return err
			}
			return output.PrintIP(os.Stdout, args[0], ip, flagJSON)
		},
	}
	c.Flags().BoolVar(&wait, "wait", false, i18n.T("flag.wait"))
	return c
}

// buildSSHArgs assembles ssh arguments: options first (the -- extras
// are options too), then the destination.
func buildSSHArgs(user, ip, port, identity string, extra []string) []string {
	var a []string
	if port != "" && port != "22" {
		a = append(a, "-p", port)
	}
	if identity != "" {
		a = append(a, "-i", identity)
	}
	a = append(a, extra...)
	return append(a, user+"@"+ip)
}

// runSSH starts ssh with the terminal attached and mirrors its exit code.
func runSSH(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		os.Exit(ee.ExitCode())
	}
	return fmt.Errorf("%s: %w", bin, err)
}

// joinCommand renders a command with arguments shell-quoted, for
// display purposes (dry-run).
func joinCommand(bin string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, bin)
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n'\"\\$&|;<>(){}*?[]#~") {
			parts = append(parts, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}
