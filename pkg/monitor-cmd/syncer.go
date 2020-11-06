package monitor

import "sync"

// CoordinatorStatus will indicate the status of all the commands
// being coordinated
type CoordinatorStatus int

const (
	// StatusIncomplete means not all commands have finished running
	StatusIncomplete CoordinatorStatus = iota
	// StatusDone means all commands have finished running
	StatusDone CoordinatorStatus = iota
)

// Coordinator uses a channel for commands to communicate their donness
// and has a mutex to prevent races against a PendingCmdCount
type Coordinator struct {
	// SyncChan can be listened to by other
	SyncChan        chan bool
	PendingCmdCount int
	sync.RWMutex
}

// NewCoordinator makes a new coordinator for a given number of cmds
func NewCoordinator() *Coordinator {
	return &Coordinator{
		SyncChan: make(chan bool),
	}
}

// FinishCommand decrements PendingCmdCount in a race safe manner
func (c *Coordinator) FinishCommand() {
	c.Lock()
	c.PendingCmdCount--
	c.Unlock()
}

// GetStatus returns if the PendingCmdCount is equal to zero
// It is race safe via a read lock
func (c *Coordinator) GetStatus() CoordinatorStatus {
	c.RLock()
	defer c.RUnlock()

	if c.PendingCmdCount == 0 {
		return StatusDone
	}
	return StatusIncomplete
}
