package i18n

// enCatalog is the English message directory. Keys must stay in sync
// with zhCatalog (enforced by TestCatalogParity).
var enCatalog = map[string]string{
	// root
	"root.short": "Control VMware virtual machines",
	"root.long":  "vmctl controls VMware virtual machines with a prlctl-style command interface.",

	// global flags
	"flag.json":    "output as JSON",
	"flag.vmrun":   "path to the vmrun binary (default: auto-detect)",
	"flag.backend": "backend driver to use",
	"flag.lang":    "message language: zh or en (default: zh)",
	"flag.profile": "configuration profile to use (default: the file's default)",

	// config / profile
	"err.config.notExist":      "config file not found: %s",
	"err.config.read":          "failed to read config file: %s",
	"err.config.parse":         "failed to parse config file: %s",
	"err.config.profile":       "profile %q not found; available: %s",
	"err.config.profileNoList": "profile %q not found and no profiles are defined",
	"err.config.envEmpty":      "environment variable %s is unset or empty; cannot read the password",

	// errors
	"err.prefix":       "Error",
	"err.notSupported": "operation not supported by this backend",

	// list / info
	"cmd.list.short": "List virtual machines",
	"cmd.info.short": "Show detailed information about a virtual machine",

	// power
	"cmd.start.short":   "Start or resume a virtual machine",
	"cmd.stop.short":    "Power off a virtual machine",
	"cmd.suspend.short": "Suspend a virtual machine to disk",
	"cmd.resume.short":  "Resume a suspended or paused virtual machine",
	"cmd.pause.short":   "Pause a running virtual machine",
	"cmd.reset.short":   "Reset a virtual machine",
	"flag.hard":         "force the operation (skip guest shutdown)",
	"flag.nogui":        "start without opening a window",

	// clone
	"cmd.clone.short": "Clone a virtual machine",
	"cmd.clone.long": `Clone a virtual machine. By default a linked clone is created
(fast; keeps a base snapshot on the source). Use --full for an
independent copy. The source must be powered off.`,
	"flag.clone.name": "name for the new virtual machine (required)",
	"flag.full":       "create an independent full copy instead of a linked clone",
	"flag.snapshot":   "base snapshot name for the clone",
	"flag.path":       "destination bundle directory or .vmx path",

	// create
	"cmd.create.short": "Create a virtual machine from an existing one",
	"cmd.create.long": `Provision a new virtual machine by cloning a source VM.
Creating a bare VM from scratch is not supported by the vmrun
backend (Fusion provides no createVM command).`,
	"flag.from":   "source virtual machine to clone from (required)",
	"flag.memory": "memory in MB for the new VM (0 = keep source)",
	"flag.cpus":   "number of CPUs for the new VM (0 = keep source)",

	// set
	"cmd.set.short":   "Change virtual machine configuration",
	"cmd.set.long":    "Change memory, CPU count or display name of a powered-off VM.",
	"err.set.flags":   "specify at least one of --name, --memory, --cpus",
	"flag.set.name":   "new display name",
	"flag.set.memory": "memory in MB",
	"flag.set.cpus":   "number of CPUs",

	// delete
	"cmd.delete.short": "Permanently delete a virtual machine and its files",
	"cmd.delete.long":  "Permanently delete a virtual machine, including all files on disk. This cannot be undone.",

	// snapshot
	"cmd.snapshot.short":        "Manage virtual machine snapshots",
	"cmd.snapshot.long":         "Work with snapshots: list, create, delete and revert.\nReverting requires the VM to be powered off.",
	"cmd.snapshot.list.short":   "List snapshots of a virtual machine",
	"cmd.snapshot.create.short": "Create a snapshot",
	"cmd.snapshot.delete.short": "Delete a snapshot",
	"cmd.snapshot.delete.long":  "Delete a snapshot. Child snapshots are re-parented by\ndefault; pass --children to delete them as well.",
	"cmd.snapshot.revert.short": "Revert a virtual machine to a snapshot",
	"flag.tree":                 "show snapshot hierarchy",
	"flag.children":             "also delete child snapshots",

	// exec
	"cmd.exec.short": "Run a command inside the guest OS",
	"cmd.exec.long": `Run a command inside the guest via VMware Tools. The command
is executed with /bin/sh, its combined output is printed and
the guest exit code becomes the vmctl exit code.
Quote the whole command to use shell features, e.g.:
  vmctl exec my-vm -- 'df -h | grep /'
Guest credentials are passed to vmrun on its command line;
prefer the VMCTL_GUEST_PASSWORD environment variable over
--password to keep them out of shell history.`,
	"flag.exec.user":     "guest user name (default: current host user)",
	"flag.exec.password": "guest password (or set VMCTL_GUEST_PASSWORD)",

	// shell
	"cmd.shell.short": "Open an SSH session to the guest",
	"cmd.shell.long": `Resolve the guest's IP via VMware Tools and start ssh.
The guest must run a reachable sshd with key or password
authentication. Options after -- are passed to ssh, e.g.:
  vmctl shell my-vm -- -v
Use --dry-run to print the ssh command instead of running it.`,
	"err.shell.args":      "accepts 1 arg(s), received %d",
	"flag.shell.user":     "login user (default: current host user)",
	"flag.shell.port":     "ssh port",
	"flag.shell.identity": "ssh private key file",
	"flag.wait":           "wait until the guest reports an IP",
	"flag.dry-run":        "print the ssh command instead of connecting",

	// ip
	"cmd.ip.short": "Print the guest IP address",
	"cmd.ip.long":  "Print the guest IP as reported by VMware Tools — handy in\nscripts: vmctl ip my-vm",

	// version
	"cmd.version.short": "Print vmctl version",
}
