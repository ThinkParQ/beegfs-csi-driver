/*
Copyright 2022 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import "sync"

// threadSafeStringLock maintains a threadsafe set of strings and provides easily consumable methods for obtaining and
// releasing a lock on a string. Use a threadSafeStringLock to ensure only one Goroutine makes use of or references a
// particular string at a any given time.
type threadSafeStringLock struct {
	rwMutex sync.RWMutex
	items   map[string]struct{}
}

func newThreadSafeStringLock() *threadSafeStringLock {
	return &threadSafeStringLock{
		items: make(map[string]struct{}),
	}
}

// obtainLockOnString locks a string for the current Goroutine and returns true if the string is not already in use by
// another Goroutine. obtainLockOnString returns false otherwise.
func (v *threadSafeStringLock) obtainLockOnString(stringToLock string) bool {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()
	if _, ok := v.items[stringToLock]; !ok {
		// stringToLock is not in map (and not in use by another Goroutine). Lock stringToLock and return success.
		v.items[stringToLock] = struct{}{}
		return true
	}
	// stringToLock is in map (and in use by another Goroutine). Return failure.
	return false
}

// releaseLockOnString releases the lock on a string.
func (v *threadSafeStringLock) releaseLockOnString(stringToUnlock string) {
	v.rwMutex.Lock()
	defer v.rwMutex.Unlock()
	delete(v.items, stringToUnlock)
}

// volumeStatus introduces a type-safe set of strings that can represent the lifecycle state of a volume.
type volumeStatus string

// Introducing ephemeral states (e.g. creating, deleting) is possible, but these states complicate the control loop
// unnecessarily (we have to revert to some previous state if we fail while creating for example). Ultimately, we only
// need to know if we have already reached a well-defined checkpoint so we can decide whether or not to continue working
// on a request.
const (
	statusCreated volumeStatus = "created"
	statusDeleted volumeStatus = "deleted"
)

// threadSafeStatusMap maintains a thread safe map of strings (volumeIDs) to well-defined volumeStatuses (e.g. created,
// deleted). threadSafeStatusMap enables a service to "remember" if it has already reached some well-defined checkpoint
// for a volume. This protects us in scenarios in which an operation takes longer than the gRPC client is willing to
// wait, but eventually completes successfully (e.g. interface fallback). The next time the gRPC client makes the same
// request, the service remembers it has already completed it and returns immediately.
type threadSafeStatusMap struct {
	rwMutex sync.RWMutex
	items   map[string]volumeStatus
}

func newThreadSafeStatusMap() *threadSafeStatusMap {
	return &threadSafeStatusMap{
		items: make(map[string]volumeStatus),
	}
}

// writeStatus safely updates the status for an existing volume or adds the status for a new volume.
func (m *threadSafeStatusMap) writeStatus(volumeID string, status volumeStatus) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	m.items[volumeID] = status
}

// readStatus safely reads the status of a volume. Like the underlying map, readStatus returns status = "" and
// ok = false if the volume status doesn't exist.
func (m *threadSafeStatusMap) readStatus(volumeID string) (status volumeStatus, ok bool) {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()
	status, ok = m.items[volumeID]
	return status, ok
}
