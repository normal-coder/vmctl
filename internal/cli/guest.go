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
	"gitee.com/normalcoder/vmctl/internal/output"
)

func newExecCmd() *cobra.Command {
	var guestUser, password string
	c := &cobra.Command{
		Use:   "exec <vm> -- <command>",
		Short: "Run a command inside the guest OS",
		Long: "Run a command inside the guest via VMware Tools. The command\n" +
			"is executed with /bin/sh, its combined output is printed and\n" +
			"the guest exit code becomes the vmctl exit code.\n" +
			"Quote the whole command to use shell features, e.g.:\n" +
			"  vmctl exec my-vm -- 'df -h | grep /'\n" +
			"Guest credentials are passed to vmrun on its command line;\n" +
			"prefer the VMCTL_GUEST_PASSWORD environment variable over\n" +
			"--password to keep them out of shell history.",
		Args: cobra.MinimumNArgs(2),
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
	c.Flags().StringVarP(&guestUser, "user", "u", "", "guest user name (default: current host user)")
	c.Flags().StringVarP(&password, "password", "p", "", "guest password (or set VMCTL_GUEST_PASSWORD)")
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
		Short: "Open an SSH session to the guest",
		Long: "Resolve the guest's IP via VMware Tools and start ssh.\n" +
			"The guest must run a reachable sshd with key or password\n" +
			"authentication. Options after -- are passed to ssh, e.g.:\n" +
			"  vmctl shell my-vm -- -v\n" +
			"Use --dry-run to print the ssh command instead of running it.",
		// Exactly one <vm> before --; anything after -- is for ssh.
		Args: func(cmd *cobra.Command, args []string) error {
			dash := cmd.Flags().ArgsLenAtDash()
			if dash < 0 {
				dash = len(args)
			}
			if dash != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", dash)
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
	c.Flags().StringVarP(&login, "user", "u", "", "login user (default: current host user)")
	c.Flags().StringVarP(&port, "port", "", "22", "ssh port")
	c.Flags().StringVarP(&identity, "identity", "i", "", "ssh private key file")
	c.Flags().BoolVar(&wait, "wait", false, "wait until the guest reports an IP")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print the ssh command instead of connecting")
	return c
}

func newIPCmd() *cobra.Command {
	var wait bool
	c := &cobra.Command{
		Use:   "ip <vm>",
		Short: "Print the guest IP address",
		Long: "Print the guest IP as reported by VMware Tools — handy in\n" +
			"scripts: vmctl ip my-vm",
		Args: cobra.ExactArgs(1),
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
	c.Flags().BoolVar(&wait, "wait", false, "wait until the guest reports an IP")
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
