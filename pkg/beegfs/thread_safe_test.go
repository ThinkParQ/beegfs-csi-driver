/*
Copyright 2022 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestThreadSafeStringLock(t *testing.T) {
	tssl := newThreadSafeStringLock()
	const numStrings = 2
	const numRoutinesPerString = 5

	// Track a map of channels indexed by the name of the string to be locked. Each channel receives true from each
	// Goroutine that successfully obtains a lock on its associated string and false from each Goroutine that does not.
	locks := map[string]chan bool{}
	for i := 0; i < numStrings; i++ {
		locks[fmt.Sprintf("string%d", i)] = make(chan bool, numRoutinesPerString)
	}

	for lockString, resultChannel := range locks {
		// Create multiple Goroutines that each sleep a random length of time before attempting to obtain the lock and
		// report success or failure to the appropriate channel.
		for i := 0; i < numRoutinesPerString; i++ {
			go func(lockString string, resultChannel chan bool) {
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
				resultChannel <- tssl.obtainLockOnString(lockString)
			}(lockString, resultChannel)
		}
	}

	// Verify that each lock was obtained one and only one time.
	for lockString, resultChannel := range locks {
		receivedTrue := false
		for i := 0; i < numRoutinesPerString; i++ {
			if <-resultChannel {
				if !receivedTrue {
					receivedTrue = true
				} else {
					t.Fatalf("expected only one successful lock on %s but got more than one", lockString)
				}
			}
		}
		if !receivedTrue {
			t.Fatalf("expected at least one successful lock on %s but got none", lockString)
		}
	}

	// Verify that releasing each lock allows it to be obtained again.
	for lockString := range locks {
		tssl.releaseLockOnString(lockString)
		if !tssl.obtainLockOnString(lockString) {
			t.Fatalf("expected to be able to relock released lock on %s but could not", lockString)
		}
	}
}

func TestThreadSafeStatusMapNoContention(t *testing.T) {
	tssm := newThreadSafeStatusMap()
	const volumeID = "volumeID"

	// Read an empty status.
	_, ok := tssm.readStatus(volumeID)
	if ok {
		t.Fatalf("expected ok to be false")
	}

	// Write a status.
	tssm.writeStatus(volumeID, statusCreated)

	// Read a non-empty status.
	status, ok := tssm.readStatus(volumeID)
	if !ok {
		t.Fatalf("expected ok to be true")
	} else if status != statusCreated {
		t.Fatalf("expected status %s to be %s", status, statusCreated)
	}
}

func TestThreadSafeStatusMapContention(t *testing.T) {
	tssm := newThreadSafeStatusMap()
	const volumeID = "volumeID"

	// Create a Goroutine to hold a write lock.
	quitLock := make(chan bool)
	go func() {
		tssm.rwMutex.Lock()
		<-quitLock // Block until we receive on quitLock.
		tssm.rwMutex.Unlock()
	}()

	// Create a Goroutine to try to read status.
	readCompleted := make(chan bool)
	go func() {
		tssm.readStatus(volumeID)
		readCompleted <- true
	}()

	// Create a Goroutine to try to write status.
	writeCompleted := make(chan bool)
	go func() {
		tssm.writeStatus(volumeID, statusCreated)
		writeCompleted <- true
	}()

	// Check if read and write completed.
	select {
	case <-readCompleted:
		t.Fatalf("did not expect read to complete while lock is held")
	default:
		// This is expected.
	}
	select {
	case <-writeCompleted:
		t.Fatalf("did not expect write to complete while lock is held")
	default:
		// This is expected.
	}

	// Cancel lock and try to read and write.
	quitLock <- true
	cancelTime := time.Now()
	select {
	case <-readCompleted:
		// This is expected.
	default:
		if time.Since(cancelTime) > 5*time.Second { // Wait a bit for the lock to be released.
			t.Fatalf("expected read to complete after lock is released")
		}
	}
	select {
	case <-writeCompleted:
		// This is expected.
	default:
		if time.Since(cancelTime) > 5*time.Second { // Wait a bit for the lock to be released.
			t.Fatalf("expected write to complete after lock is released")
		}
	}
}
