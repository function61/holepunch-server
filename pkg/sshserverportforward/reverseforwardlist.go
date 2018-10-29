package sshserverportforward

import (
	"fmt"
	"sync"
)

type forwardList struct {
	sync.Mutex
	reverseCancellations map[string]chan bool
}

func (f *forwardList) add(cfm channelForwardMsg) *chan bool {
	f.Lock()
	defer f.Unlock()

	cancellationKey := toCancellationKey(cfm)

	if _, exists := f.reverseCancellations[cancellationKey]; exists {
		return nil
	}

	cancelCh := make(chan bool, 1)
	f.reverseCancellations[cancellationKey] = cancelCh

	return &cancelCh
}

func (f *forwardList) cancel(cfm channelForwardMsg) bool {
	f.Lock()
	defer f.Unlock()

	cancellationKey := toCancellationKey(cfm)

	cancelCh, exists := f.reverseCancellations[cancellationKey]
	if !exists {
		return false
	}

	cancelCh <- true
	delete(f.reverseCancellations, cancellationKey)

	return true
}

func toCancellationKey(cfm channelForwardMsg) string {
	return fmt.Sprintf("%s:%d", cfm.Addr, cfm.Rport)
}
