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
	"flag.vmcli":   "path to the vmcli binary (default: auto-detect; requires Fusion 13.5+)",
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

	// resolve (shared reference resolution wording)
	"err.resolve.empty":          "reference must not be empty",
	"err.resolve.notFound":       "virtual machine %q not found (known: %s)",
	"err.resolve.ambiguous":      "ambiguous reference %q matches %d VMs",
	"err.resolve.ambiguousNames": "ambiguous reference %q matches: %s",

	// vmcli (snapshot uid fallback)
	"err.vmcli.notFound": "vmcli not found (requires VMware Fusion 13.5+; set --vmcli or VMCTL_VMCLI)",

	// vmrun backend
	"err.name.empty":           "name must not be empty",
	"err.name.invalid":         "invalid name %q: must not contain /, \\, quotes or control characters",
	"err.name.exists":          "a VM named %q already exists",
	"err.vmrun.notFound":       "vmrun not found (install VMware Fusion/Workstation or set VMCTL_VMRUN)",
	"err.vmrun.runningFor":     "cannot %s while the VM is running: power it off first",
	"err.vmrun.notRunningFor":  "cannot %s while the VM is not running: power it on first",
	"err.vmrun.powerOnTimeout": "timed out waiting for %q to power on",
	"err.vmrun.offUseStart":    "%q is powered off, use start",
	"err.vmrun.exitZero":       "vmrun reported an error with exit code 0",
	"err.dest.exists":          "destination already exists: %s",
	"err.dest.notDir":          "destination is not a directory: %s",
	"err.dest.notEmpty":        "destination directory is not empty: %s",
	"err.clone.regFail":        "cloned to %s, but inventory registration failed: %w",
	"err.clone.configFail":     "cloned to %s, but post-clone configuration failed: %w",
	"err.clone.noFrom":         "this backend cannot create a bare VM; pass --from to clone an existing one",
	"err.set.memoryPositive":   "memory must be greater than 0 MB",
	"err.set.cpusPositive":     "cpus must be greater than 0",
	"err.set.renameFail":       "updated %s, but inventory rename failed: %w",
	"err.delete.cleanupFail":   "deleted, but inventory cleanup failed: %w",
	"err.vmx.utf16":            "%s: UTF-16 encoded vmx is not supported for editing",
	"err.snapshot.invalidName": "invalid snapshot name: %w",

	// output
	"output.noVMs":       "No virtual machines found.",
	"output.noSnapshots": "No snapshots for %s.",
	"output.ok":          "ok",

	// vsphere backend
	"err.vsphere.endpoint":         "vsphere backend requires an endpoint; configure it in a profile",
	"err.vsphere.user":             "vsphere backend requires a user name",
	"err.vsphere.connect":          "failed to connect to %s: %s",
	"err.vsphere.notRunning":       "virtual machine is not powered on",
	"err.vsphere.stopTimeout":      "timed out waiting for the VM to power off (60s); VMware Tools may be missing or not running",
	"err.vsphere.offForResume":     "virtual machine is powered off; use start instead",
	"err.vsphere.noPause":          "vSphere has no pause operation; use suspend to suspend to disk",
	"err.vsphere.mustBeOff":        "virtual machine must be powered off",
	"err.vsphere.createNoFrom":     "the vsphere backend cannot create a bare VM; pass --from to clone an existing one",
	"err.vsphere.linkedNoSnapshot": "linked clone needs a base snapshot, but source VM %q has none",
	"err.vsphere.cloneFolder":      "clone target folder not found: %s",
	"err.vsphere.cloneNotFolder":   "clone target is not a folder: %s",
	"err.vsphere.cloneResult":      "clone task returned no new VM",
	"err.clone.name":               "a name for the new virtual machine is required",
	"err.vsphere.noIP":             "guest has not reported an IP address yet (is VMware Tools running?)",
	"err.exec.empty":               "command must not be empty",

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
	"cmd.snapshot.short":            "Manage virtual machine snapshots",
	"cmd.snapshot.long":             "Work with snapshots: list, create, delete and revert.\nReverting requires the VM to be powered off.",
	"cmd.snapshot.list.short":       "List snapshots of a virtual machine",
	"cmd.snapshot.create.short":     "Create a snapshot",
	"cmd.snapshot.delete.short":     "Delete a snapshot",
	"cmd.snapshot.delete.long":      "Delete a snapshot. Child snapshots are re-parented by\ndefault; pass --children to delete them as well.",
	"cmd.snapshot.revert.short":     "Revert a virtual machine to a snapshot",
	"flag.tree":                     "show snapshot hierarchy",
	"flag.children":                 "also delete child snapshots",
	"err.snapshot.needName":         "a snapshot name is required",
	"err.snapshot.ambiguous":        "snapshot name %q is ambiguous, candidates: %s (pick one with <name#uid>)",
	"err.snapshot.ambiguousNoVmcli": "snapshot name %q is ambiguous and vmcli cannot list candidates;\nset --vmcli to enable <name#uid> references",

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

	// completion
	"cmd.completion.short": "Generate shell autocompletion scripts",
	"cmd.completion.long": `Generate a shell autocompletion script for vmctl,
covering commands, global flags (--backend/--profile/--lang)
and virtual machine names. See each sub-command's help for
how to load the generated script.`,
	"cmd.completion.bash":       "Generate the autocompletion script for bash",
	"cmd.completion.zsh":        "Generate the autocompletion script for zsh",
	"cmd.completion.fish":       "Generate the autocompletion script for fish",
	"cmd.completion.powershell": "Generate the autocompletion script for powershell",
}
