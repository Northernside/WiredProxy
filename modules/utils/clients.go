package utils

import (
	"sync"

	"wired.rip/wiredutils/protocol"
)

type Node struct {
	Key  string
	Arch string
}

type Client struct {
	Key  string
	Conn protocol.Conn
	Data Node
}

var (
	Clients      = make(map[string]Client)
	ClientsMutex = &sync.Mutex{}
)

func AddClient(key string, conn protocol.Conn, data Node) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	Clients[conn.RemoteAddr().String()] = Client{
		Key:  key,
		Conn: conn,
		Data: data,
	}
}

func RemoveClient(key string) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	delete(Clients, key)
}

func FindClient(key string) (protocol.Conn, Node, bool) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	if client, ok := Clients[key]; ok {
		return client.Conn, client.Data, true
	}

	return protocol.Conn{}, Node{}, false
}

func GetClients() map[string]protocol.Conn {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	clients := make(map[string]protocol.Conn)
	for key, client := range Clients {
		clients[key] = client.Conn
	}

	return clients
}
