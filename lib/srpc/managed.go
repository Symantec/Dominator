package srpc

import (
	"crypto/tls"
	"sync"
	"syscall"
	"time"
)

type clientKey struct {
	network string
	address string
}

var (
	maxConnections      int
	connectionSemaphore chan struct{}
	lock                sync.Mutex
	limitsSet           bool
	keyToInUseClients   = make(map[clientKey]*Client)
	usedClientToKey     = make(map[*Client]clientKey)
	keyToFreeClients    = make(map[clientKey]*Client)
)

func getConnectionLimit() int {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		return 900
	}
	maxConnAttempts := rlim.Cur - 50
	maxConnAttempts = (maxConnAttempts / 100)
	if maxConnAttempts < 1 {
		maxConnAttempts = 1
	} else {
		maxConnAttempts *= 100
	}
	return int(maxConnAttempts)
}

func getHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration, wait bool) (*Client, error) {
	// Delay setting of internal limits to allow application code to increase the
	// limit on file descriptors first.
	if !limitsSet {
		lock.Lock()
		if !limitsSet {
			maxConnections = getConnectionLimit()
			connectionSemaphore = make(chan struct{}, maxConnections)
			limitsSet = true
		}
		lock.Unlock()
	}
	// Grab a connection slot (the right to create a Client).
	if wait {
		connectionSemaphore <- struct{}{}
	} else {
		select {
		case connectionSemaphore <- struct{}{}:
		default:
			return nil, nil
		}
	}
	key := clientKey{network: network, address: address}
	lock.Lock()
	defer lock.Unlock()
	if _, ok := keyToInUseClients[key]; ok {
		panic("already have gotten " + network + "," + address)
	}
	// First try and re-use an existing free Client.
	if client, ok := keyToFreeClients[key]; ok {
		delete(keyToFreeClients, key)
		keyToInUseClients[key] = client
		usedClientToKey[client] = key
		client.free = false
		return client, nil
	}
	// Will have to create a new Client. May have to create or re-use a slot.
	if len(keyToInUseClients)+len(keyToFreeClients) >= maxConnections {
		deleted := false
		// Need to grab a free Client and close. Be lazy and do a random pick.
		for key, client := range keyToFreeClients {
			delete(keyToFreeClients, key)
			client.isManaged = false
			client.Close()
			deleted = true
			break
		}
		if !deleted {
			panic("no free Client to Close()")
		}
	}
	lock.Unlock()
	client, err := dialHTTP(network, address, tlsConfig, timeout)
	lock.Lock()
	if err != nil {
		<-connectionSemaphore // Free up a slot for someone else.
		return nil, err
	}
	client.isManaged = true
	keyToInUseClients[key] = client
	usedClientToKey[client] = key
	client.free = false
	return client, nil
}

func (client *Client) put(remove bool) {
	if client.closed {
		return
	}
	client.release(remove)
	<-connectionSemaphore // Free up a slot for someone else.
}

func (client *Client) release(remove bool) {
	if !client.isManaged {
		panic("client not managed")
	}
	if client.free {
		panic("client has already been Put()")
	}
	lock.Lock()
	defer lock.Unlock()
	key, ok := usedClientToKey[client]
	if !ok {
		panic("client not found on usedClientToKey map")
	}
	delete(keyToInUseClients, key)
	delete(usedClientToKey, client)
	if !remove {
		keyToFreeClients[key] = client
	}
	client.free = true
}
