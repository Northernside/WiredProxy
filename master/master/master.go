package master

import (
	"bufio"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
	"wiredmaster/routes"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/packet"
	"wired.rip/wiredutils/protocol"
	"wired.rip/wiredutils/terminal"
)

var clients = make(map[string]protocol.Conn)

func Run() {
	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)

	go startHttpServer()
	go routeUpdater()
	loadWiredKeyPair()
	startServer()
}

func startHttpServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not found"}`))
			return
		}

		routes.IndexRoute(w, r)
	})

	customHandler("/api/routes", routes.GetRoutes, http.MethodGet)
	customHandler("/api/routes/add", routes.AddRoute, http.MethodGet)
	customHandler("/api/routes/remove", routes.RemoveRoute, http.MethodDelete)
	customHandler("/api/node/update", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// send update packet

		folder := r.URL.Query().Get("folder")
		gitPullCmd := exec.Command("git", "pull")
		gitPullCmd.Dir = folder
		err := gitPullCmd.Run()
		if err != nil {
			log.Println("Error running git pull:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "Internal server error"}`))
			return
		}

		for _, client := range clients {
			sendBinaryUpdate(client, folder)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Update packet sent"}`))
	}, http.MethodGet)

	http.HandleFunc("/api/connect/publickey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		w.Write(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&wiredKey.PublicKey),
		}))
	})

	log.Println("HTTP server listening on 127.0.0.1:37421")
	http.ListenAndServe("127.0.0.1:37421", nil)
}

func customHandler(path string, handler http.HandlerFunc, method string) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"message": "Method not allowed"}`))
			return
		}

		handler(w, r)
	})
}

func startServer() {
	server, err := net.Listen("tcp", ":37420")
	if err != nil {
		panic(err)
	}

	log.Println("Communication server listening on *:37420")
	for {
		conn, err := server.Accept()
		if err != nil {
			continue
		}

		go handleConnection(protocol.NewConn(conn, wiredKey, nil))
	}
}

func handleConnection(conn *protocol.Conn) {
	defer conn.Close()

	var pp protocol.Packet
	err := pp.Read(conn)
	if err != nil {
		return
	}

	var sharedSecret []byte
	err = protocol.DecodePacket(pp.Data, &sharedSecret)
	if err != nil {
		log.Println("Error decoding shared secret:", err)
		return
	}

	err = conn.EnableEncryption(sharedSecret)
	if err != nil {
		log.Println("Error enabling encryption:", err)
		return
	}

	err = conn.SendPacket(packet.Id_Ready, nil)
	if err != nil {
		log.Println("Error sending ready packet:", err)
		return
	}

	// add client to clients map
	clients[string(conn.Address)] = *conn

	for {
		var pp protocol.Packet
		err := pp.Read(conn)
		if err != nil {
			// log.Println("Error reading packet:", err)
			continue
		}

		// log.Println("Received packet:", pp.ID)
		// log.Println("Data:", string(pp.Data))

		switch pp.ID {
		case packet.Id_Hello:
			log.Printf("Received hello packet at %s\n", time.Now().Format("15:04:05"))
			var hello packet.Hello
			err := protocol.DecodePacket(pp.Data, &hello)
			if err != nil {
				log.Println("Error decoding hello packet:", err)
				continue
			}

			log.Printf("Client %s.%s connected with version %s\n", hello.Key, config.GetWiredHost(), hello.Version)

			log.Println("Client hash:", string(hello.Hash))
			log.Println("Current node hash:", config.GetCurrentNodeHash())

			if string(hello.Hash) != config.GetCurrentNodeHash() {
				log.Println("Node hash mismatch, sending update packet")
				sendBinaryUpdate(*conn, "")
			}

			routes := config.GetRoutes()

			// send routes packet
			err = conn.SendPacket(packet.Id_Routes, packet.Routes{
				Routes: routes,
			})

			if err != nil {
				log.Println("Error sending routes packet:", err)
				continue
			}
		case packet.Id_Ping:
			err = conn.SendPacket(packet.Id_Pong, nil)
			if err != nil {
				log.Println("Error sending pong packet:", err)
				continue
			}

			// log.Println("Sent pong")
		}
	}
}

func routeUpdater() {
	for {
		<-routes.SignalChannel

		log.Println("Sending routes packet to all clients")

		pData, err := protocol.MarshalPacket(packet.Id_Routes, packet.Routes{
			Routes: config.GetRoutes(),
		})
		if err != nil {
			log.Println("Error encoding routes packet:", err)
			continue
		}

		for _, client := range clients {
			log.Println("Sending routes packet to", client.Address)
			_, err = client.Write(pData)
			if err != nil {
				log.Println("Error sending routes packet to", client.Address, ":", err)
				continue
			}
		}
	}
}

func sendBinaryUpdate(client protocol.Conn, _folder string) {
	log.Println("Sending update packet to", client.Address)

	folder := _folder
	if _folder == "" {
		folder = "../node"
	}

	// check if folder exists
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		log.Println("Error checking folder:", err)
		return
	}

	// check if folder/mod.go exists
	if _, err := os.Stat(folder + "/go.mod"); os.IsNotExist(err) {
		log.Println("Error checking go.mod:", err)
		return
	}

	// read first line of mod.go
	// and check if its "module wirednode"

	moduleFile, err := os.Open(folder + "/go.mod")
	if err != nil {
		log.Println("Error opening go.mod:", err)
		return
	}

	// split by lines
	scanner := bufio.NewScanner(moduleFile)
	scanner.Scan()
	if scanner.Text() != "module wirednode" {
		log.Println("Error: go.mod is not a wirednode module")
		return
	}

	goBuildCmd := exec.Command("go", "build")
	goBuildCmd.Dir = folder
	err = goBuildCmd.Run()
	if err != nil {
		log.Println("Error building module:", err)
		return
	}

	filename := folder + "/wirednode"
	err = client.SendFile("upgrade", filename, packet.Id_BinaryData, packet.Id_BinaryEnd)
	if err != nil {
		log.Println("Error sending update packet to", client.Address, ":", err)
		return
	}
}
