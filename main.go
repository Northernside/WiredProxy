package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	handshake "wiredproxy/utils"
)

var (
	serverHost = os.Args[1]
	serverPort = os.Args[2]
	proxyPort  = os.Args[3]
)

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", proxyPort))
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Listening on %s:%s\n", "127.0.0.1", proxyPort)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(clientConn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	handshakePacket, err := handshake.ReadPacket(clientConn)
	if err != nil {
		fmt.Println("Error reading handshake packet:", err)
		return
	}

	editedHandshakePacket := *handshakePacket
	editedHandshakePacket.Hostname = handshake.String(serverHost)

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverHost, serverPort))
	if err != nil {
		fmt.Println("Error connecting to Minecraft server:", err)
		return
	}
	defer serverConn.Close()

	// Send handshake packet to server
	err = handshake.WritePacket(editedHandshakePacket, serverConn)
	if err != nil {
		fmt.Println("Error sending handshake packet to server:", err)
		return
	}

	// C->S
	go io.Copy(clientConn, serverConn)

	// S->C
	io.Copy(serverConn, clientConn)
}

func logPacket(data []byte) {
	fmt.Println(hex.Dump(data))
}
