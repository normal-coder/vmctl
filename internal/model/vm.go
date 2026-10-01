// Package model defines backend-agnostic types shared by all drivers.
package model

// State is the power state of a virtual machine.
type State string

const (
	StateOn        State = "on"
	StateOff       State = "off"
	StateSuspended State = "suspended"
	StatePaused    State = "paused"
	StateUnknown   State = "unknown"
)

// VM is the backend-agnostic view of a virtual machine.
type VM struct {
	// ID is a stable identifier within its backend
	// (inventory UUID for vmrun, ManagedObjectReference for vsphere).
	ID       string `json:"id"`
	Name     string `json:"name"`
	State    State  `json:"state"`
	Path     string `json:"path,omitempty"` // vmx file path (local backends)
	CPUs     int    `json:"cpus,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
	GuestOS  string `json:"guest_os,omitempty"`
}

// VMInfo carries per-VM details beyond the list view.
type VMInfo struct {
	VM
	ToolsState string `json:"tools_state,omitempty"`
	GuestIP    string `json:"guest_ip,omitempty"`
}
