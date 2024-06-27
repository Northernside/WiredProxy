package utils

import (
	"sync"

	"wired.rip/wiredutils/protocol"
)

var (
	Clients      = make(map[string]protocol.Conn)
	ClientsMutex = &sync.Mutex{}
)

func AddClient(key string, conn protocol.Conn) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	Clients[key] = conn
}

func RemoveClient(key string) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	delete(Clients, key)
}

func FindClient(key string) (protocol.Conn, bool) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	conn, ok := Clients[key]
	return conn, ok
}

func GetClients() map[string]protocol.Conn {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	return Clients
}
