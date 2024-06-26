package node

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
	"wirednode/protocol"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/packet"
	prtcl "wired.rip/wiredutils/protocol"
	"wired.rip/wiredutils/resolver"
	"wired.rip/wiredutils/utils"
)

func Run() {
	connectToMaster()
}

var (
	wiredPub *rsa.PublicKey
	master   *prtcl.Conn
)

func connectToMaster() {
	config.Init()
	loadPublicKey()
	dialMaster()
	go handleMasterConnection()
	startProxyServer()
}

func dialMaster() {
	var c net.Conn
	var err error
	failedAttempts := 0

	remoteAddr, err := resolver.ResolveWired(config.GetWiredHost())
	if err != nil {
		panic("Error resolving wired addr:" + err.Error()) // expected to recover
	}

	for {
		c, err = net.Dial("tcp", remoteAddr.String())
		if err == nil {
			failedAttempts = 0
			break
		}

		failedAttempts++
		if failedAttempts == 10 {
			log.Println("10 connections attempts were unsuccessful, errors are no longer printed.")
		}

		if failedAttempts < 10 {
			log.Println(err)
			log.Println("Reconnecting in s ...")
		}

		time.Sleep(5 * time.Second)
	}

	master = prtcl.NewConn(c, nil, wiredPub)
}

func handleMasterConnection() {
	sharedSecret := []byte(utils.GenerateString(16))
	fmt.Println("Generated shared secret:", string(sharedSecret))
	err := master.SendPacket(packet.Id_SharedSecret, sharedSecret)
	if err != nil {
		log.Println("Error sending shared secret:", err)
		return
	}

	master.EnableEncryption(sharedSecret)

	master.SendPacket(packet.Id_Hello, packet.Hello{
		Key:     config.GetSystemKey(),
		Version: "1.0.0",
		Hash:    []byte("test"),
	})

	go func() {
		for {
			err := master.SendPacket(packet.Id_Ping, nil)
			if err != nil {
				log.Println("Error sending ping:", err)
				return
			}

			// fmt.Println("Sent ping")
			time.Sleep(10 * time.Second)
		}
	}()

	for {
		var pp prtcl.Packet
		err := pp.Read(master)
		if err != nil {
			// fmt.Println("Error reading packet:", err)
			continue
		}

		// fmt.Println("Received packet:", pp.ID)
		// fmt.Println("Data:", pp.Data)

		switch pp.ID {
		case packet.Id_Ready:
			fmt.Printf("Received ready at %s\n", time.Now())
		case packet.Id_Pong:
			// fmt.Println("Received pong")
		case packet.Id_Routes:
			fmt.Printf("Received routes at %s\n", time.Now())

			var routes packet.Routes
			prtcl.DecodePacket(pp.Data, &routes)

			for _, route := range routes.Routes {
				fmt.Printf("Received route: %s:%s pointing to %s:%s (%s)\n", route.ProxyDomain, route.ProxyPort, route.ServerHost, route.ServerPort, route.RouteId)
			}

			config.SetRoutes(routes.Routes)
		}
	}
}

func loadPublicKey() {
	var err error
	for {
		wiredPub, err = initWiredConnect()
		if err != nil {
			log.Println("Error init WiredConnect:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}
}

func initWiredConnect() (*rsa.PublicKey, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://master.%s/api/connect/publickey", config.GetWiredHost()), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		log.Println("Invalid status-code")
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("Received public key:\n", string(body))

	publicKey, err := decodePEMToPublicKey(string(body))
	return publicKey, err
}

func decodePEMToPublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		fmt.Println("Invalid Public key")
		os.Exit(1)
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func startProxyServer() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Proxy server started on :25565")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}

		go handleMinecraftConnection(clientConn)
	}
}

func handleMinecraftConnection(clientConn net.Conn) {
	defer clientConn.Close()

	var handshakePacket protocol.HandshakePacket
	err := handshakePacket.ReadFrom(clientConn)
	if err != nil {
		fmt.Println("Error reading handshake packet:", err)
		sendErrorScreen(clientConn, 2)
		return
	}

	route, ok := config.GetRouteByProxyDomain(string(handshakePacket.Hostname))
	if !ok {
		fmt.Printf("Route not found for %s (Client IP: %s)\n", handshakePacket.Hostname, clientConn.RemoteAddr().String())
		sendErrorScreen(clientConn, 1)
		return
	}

	handshakePacket.Hostname = protocol.String(route.ServerHost)

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", route.ServerHost, route.ServerPort))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		sendErrorScreen(clientConn, 0)
		return
	}
	defer serverConn.Close()

	// Send handshake packet to server
	err = handshakePacket.WriteTo(serverConn)
	if err != nil {
		fmt.Println("Error sending handshake packet to server:", err)
		return
	}

	// C->S
	go io.Copy(clientConn, serverConn)

	// S->C
	io.Copy(serverConn, clientConn)
}

func sendErrorScreen(clientConn net.Conn, errorType int) {
	var statusRequest protocol.Packet

	statusRequest.ReadFrom(clientConn)

	var statusResponse protocol.StatusResponse

	text := "§8[§7Wired§8] §c"
	versionName := "§4"
	if errorType == 0 {
		text += "Server is offline"
		versionName += "Offline"
	} else if errorType == 1 {
		text += "Route not found"
		versionName += "Not found"
	} else {
		text += "Network failure"
		versionName += "Failure"
	}

	newResponse := protocol.StatusResponseJSON{
		Version: protocol.Version{
			Name:     versionName,
			Protocol: 0,
		},
		Players: protocol.Players{
			Max:    0,
			Online: 0,
		},
		Description: protocol.Description{
			Text: text,
		},
	}

	n, err := json.Marshal(newResponse)
	if err != nil {
		panic(err)
	}

	statusResponse.Status = protocol.String(n)
	statusResponse.WriteTo(clientConn)
}
