## [0.2.0] - 2026-10-05

### 🚀 Features
- Localize usage template labels and flag default suffix
- Localize completion command help and flag error fallback
- Localize output table headers and backend error messages

## [0.1.0] - 2026-10-04

### 🚀 Features
- Add unified VM model and driver interface
- Add vmrun driver with list, info and power operations
- Add cobra cli with table and json output
- Extend driver interface with lifecycle operations
- Add clone, create, set and delete to vmrun driver
- Add clone, create, set and delete cli commands
- Extend driver interface with snapshots and guest exec
- Add snapshot, guest exec and async start to vmrun driver
- Add snapshot, exec, shell and ip cli commands
- Add i18n framework with zh and en message catalogs
- Add profile config with --profile flag and flag merge
- Add vsphere driver with inventory and power ops
- Add vsphere lifecycle and snapshot ops
- Add vsphere guest exec and ip
- Add snapshot uid references via vmcli fallback
- Add shell completion for flags and vm references
- Localize cobra arg and pflag flag errors
- Add snapshot uid column to list output
- Add version json output
- Split exit codes for not found and unsupported


### 🐛 Bug Fixes
- Localize remaining hardcoded strings
- Localize unknown subcommand and builtin help gaps


### 🚜 Refactor
- Unify exit codes and usage errors


### 📚 Documentation
- Add readme with usage and roadmap
- Update readme for m2 lifecycle commands
- Update readme for m3
- Add vmcli snapshot fallback to roadmap
- Document vsphere backend, profiles and language selection
- Document uid refs, completion and M5
- Document homebrew cask install and release flow

