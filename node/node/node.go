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
	"sync"
	"syscall"
	"time"
	"wirednode/protocol"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/packet"
	prtcl "wired.rip/wiredutils/protocol"
	"wired.rip/wiredutils/resolver"
	"wired.rip/wiredutils/terminal"
	"wired.rip/wiredutils/utils"
)

func Run() {
	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)
	log.Printf("Connecting to master.%s...\n", config.GetWiredHost())

	connectToMaster()
}

var (
	wiredPub      *rsa.PublicKey
	master        *prtcl.Conn
	binaryDataMux = &sync.Mutex{}
	binaryData    = make(map[string]*[][]byte)
)

func connectToMaster() {
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
		// c, err = net.Dial("tcp", "127.0.0.1:37420")
		if err == nil {
			failedAttempts = 0
			break
		}

		failedAttempts++
		if failedAttempts < 10 {
			log.Println(err)
			log.Printf("Failed to connect to master.%s, retrying in 5 seconds...\n", config.GetWiredHost())
		} else {
			log.Fatalf("failed to connect to master.%s after 10 attempts, exiting...\n", config.GetWiredHost())
		}

		time.Sleep(5 * time.Second)
	}

	master = prtcl.NewConn(c, nil, wiredPub)
}

func handleMasterConnection() {
	sharedSecret := []byte(utils.GenerateString(16))
	err := master.SendPacket(packet.Id_SharedSecret, sharedSecret)
	if err != nil {
		log.Fatalf("error sending shared secret: %s\n", err)
	}

	master.EnableEncryption(sharedSecret)
	log.Println("Secure connection established")

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
			}

			time.Sleep(10 * time.Second)
		}
	}()

	for {
		var pp prtcl.Packet
		err := pp.Read(master)
		if err != nil {
			// log.Println("Error reading packet:", err)
			continue
		}

		switch pp.ID {
		case packet.Id_Ready:
			log.Printf("Received ready packet at %s\n", time.Now().Format("15:04:05"))
		case packet.Id_Pong:
			// log.Printf("Received pong packet at %s\n", time.Now())
		case packet.Id_Routes:
			// log.Printf("Received routes packet at %s\n", time.Now())

			var routes packet.Routes
			prtcl.DecodePacket(pp.Data, &routes)

			for _, route := range routes.Routes {
				log.Printf("Received route: %s:%s pointing to %s:%s (%s)\n", route.ProxyDomain, route.ProxyPort, route.ServerHost, route.ServerPort, route.RouteId)
			}

			config.SetRoutes(routes.Routes)
		case packet.Id_BinaryData:
			log.Printf("Received binary data packet at %s\n", time.Now().Format("15:04:05"))
			var bd prtcl.BinaryData
			err := prtcl.DecodePacket(pp.Data, &bd)
			if err != nil {
				log.Println("Error decoding binary data:", err)
				continue
			}

			binaryDataMux.Lock()
			data, ok := binaryData[bd.Label]
			if !ok {
				binaryData[bd.Label] = &[][]byte{bd.Data}
				binaryDataMux.Unlock()
				continue
			}

			*binaryData[bd.Label] = append(*data, bd.Data)
			binaryDataMux.Unlock()
		case packet.Id_BinaryEnd:
			log.Printf("Received binary end packet at %s\n", time.Now().Format("15:04:05"))
			var bd prtcl.BinaryData
			err := prtcl.DecodePacket(pp.Data, &bd)
			if err != nil {
				log.Println("Error decoding binary data:", err)
				continue
			}

			func() {
				binaryDataMux.Lock()
				defer binaryDataMux.Unlock()

				data, ok := binaryData[bd.Label]
				if !ok {
					log.Println("Error label is not available:", bd.Label)
					return
				}
				defer delete(binaryData, bd.Label)

				if bd.Label == "upgrade" {
					err = upgrade(data)
					if err != nil {
						log.Println("Error upgrading binary:", err)
					}

					return
				}

				file, err := os.Create("BD_" + bd.Label)
				if err != nil {
					log.Println("Error creating file:", err)
					return
				}
				defer file.Close()

				for _, data := range *data {
					file.Write(data)
				}

				log.Printf("Wrote binary data to %s\n", file.Name())
			}()
		}
	}
}

func upgrade(data *[][]byte) error {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %s", err)
		return err
	}

	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(exePath)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(exePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode().Perm())
	if err != nil {
		log.Fatalf("Failed to create file: %s", err)
		return err
	}
	defer file.Close()

	for _, slice := range *data {
		_, err := file.Write(slice)
		if err != nil {
			log.Printf("Failed to write to file: %s\n", err)
		}
	}

	log.Println("Replaced binary")
	file.Close()
	err = restartSelf()
	if err != nil {
		log.Printf("Failed to restart self: %s", err)
		return err
	}

	return nil
}

func restartSelf() error {
	log.Println("Restarting ...")
	self, err := os.Executable()
	if err != nil {
		log.Println("Error getting executable path:", err)
		return err
	}

	args := os.Args
	env := os.Environ()

	return syscall.Exec(self, args, env)
}

func loadPublicKey() {
	var err error
	for {
		wiredPub, err = requestPublicKey()
		if err != nil {
			log.Println("Error requesting public key:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}
}

func requestPublicKey() (*rsa.PublicKey, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://master.%s/api/connect/publickey", config.GetWiredHost()), nil)
	// req, err := http.NewRequest("GET", "http://127.0.0.1:37421/api/connect/publickey", nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	publicKey, err := decodePEMToPublicKey(string(body))
	return publicKey, err
}

func decodePEMToPublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		log.Fatalf("failed to decode PEM block containing public key")
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func startProxyServer() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		log.Fatal("error starting minecraft proxy server:", err)
	}
	defer listener.Close()

	log.Println("Minecraft proxy server listening on :25565")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
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
		log.Println("error reading handshake packet:", err)
		sendErrorScreen(clientConn, 2)
		return
	}

	route, ok := config.GetRouteByProxyDomain(string(handshakePacket.Hostname))
	if !ok {
		log.Printf("Route not found for %s (Client IP: %s)\n", handshakePacket.Hostname, clientConn.RemoteAddr().String())
		sendErrorScreen(clientConn, 1)
		return
	}

	handshakePacket.Hostname = protocol.String(route.ServerHost)

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", route.ServerHost, route.ServerPort))
	if err != nil {
		log.Println("error connecting to server:", err)
		sendErrorScreen(clientConn, 0)
		return
	}
	defer serverConn.Close()

	// Send handshake packet to server
	err = handshakePacket.WriteTo(serverConn)
	if err != nil {
		log.Println("error writing handshake packet to server:", err)
		return
	}

	// C->S
	go copyData(clientConn, serverConn)

	// S->C
	copyData(serverConn, clientConn)
}

func copyData(src net.Conn, dst net.Conn) {
	// copy and log data
	buf := make([]byte, 4096)

	for {
		// src conn
		n, err := src.Read(buf)
		if err != nil {
			return
		}

		// dst conn
		_, err = dst.Write(buf[:n])
		if err != nil {
			return
		}
	}
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
