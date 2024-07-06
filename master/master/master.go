package master

import (
	"bufio"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"wiredmaster/routes"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/jwt"
	"wired.rip/wiredutils/packet"
	"wired.rip/wiredutils/protocol"
	"wired.rip/wiredutils/sqlite"
	"wired.rip/wiredutils/terminal"
	"wired.rip/wiredutils/utils"
)

func Run() {
	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)

	go startHttpServer()
	go routeUpdater()

	sqlite.Init()
	jwt.Init()

	updateRoles()
	loadWiredKeyPair()
	startServer()
}

func updateRoles() {
	adminId := config.GetAdminDiscordId()
	if adminId == "" {
		log.Println("No admin assigned in config.json yet")
		return
	}

	// check if admin exists in database
	_, _, _, _, role, err := sqlite.GetUser("discord_id", adminId)
	if err != nil {
		log.Println("Error checking if admin exists in database:", err)
		return
	}

	if role == "" {
		log.Println("Admin not yet in database. Consider signing in with Discord soon.")
		return
	}

	if role != "admin" {
		log.Println("Admin ID found in database, but role is not admin. Changing role to admin.")
		err = sqlite.ChangeUserRole(adminId, "admin")
		if err != nil {
			log.Println("Error changing user role to admin:", err)
			return
		}
	}
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

	userHandler("/api/routes", routes.GetRoutes, http.MethodGet)
	userHandler("/api/nodes", routes.GetNodes, http.MethodGet)
	userHandler("/api/users", routes.GetUsers, http.MethodGet)
	adminHandler("/api/users/role", routes.ChangeUserRole, http.MethodGet)
	adminHandler("/api/routes/add", routes.AddRoute, http.MethodGet)
	adminHandler("/api/routes/remove", routes.RemoveRoute, http.MethodDelete)
	adminHandler("/api/node/set-hash", routes.SetNodeHash, http.MethodGet)
	adminHandler("/api/node/disconnect", routes.DisconnectNode, http.MethodGet)
	adminHandler("/api/node/update", func(w http.ResponseWriter, r *http.Request) {
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

		clients := utils.GetClients()
		for _, client := range clients {
			sendBinaryUpdate(client, folder)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Update packet sent"}`))
	}, http.MethodGet)

	customHandler("/api/auth/discord", routes.AuthDiscord, http.MethodGet)
	customHandler("/api/auth/discord/callback", routes.AuthDiscordCallback, http.MethodGet)

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

func adminHandler(path string, handler http.HandlerFunc, method string) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"message": "Method not allowed"}`))
			return
		}

		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
			return
		}

		token := authorization[7:]
		claims, err := jwt.ValidateToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
			return
		}

		if claims["role"] != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
			return
		}

		handler(w, r)
	})
}

func userHandler(path string, handler http.HandlerFunc, method string) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"message": "Method not allowed"}`))
			return
		}

		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
			return
		}

		token := authorization[7:]
		claims, err := jwt.ValidateToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
			return
		}

		if claims["role"] == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
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
	key := conn.RemoteAddr().String()
	defer func() {
		_ = conn.Close()

		_, ok := utils.FindClient(key)
		if !ok {
			return
		}

		log.Printf("Node %s.%s disconnected at %s\n", key, config.GetWiredHost(), time.Now().Format("15:04:05"))
		utils.RemoveClient(key)
	}()

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

	for {
		var pp protocol.Packet
		err := pp.Read(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "connection reset by peer") {
				return
			}

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

			key = hello.Key
			log.Printf("Client %s.%s connected with version %s\n", hello.Key, config.GetWiredHost(), hello.Version)

			// add client to clients map
			utils.AddClient(hello.Key, *conn)

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
		case packet.Id_PlayerAdd:
			var player protocol.Player
			err := protocol.DecodePacket(pp.Data, &player)
			if err != nil {
				log.Println("Error decoding player add packet:", err)
				continue
			}

			utils.AddPlayer(player)
			log.Printf("Player %s (%s) joined %s with protocol version %d on %s.%s\n", player.Name, player.UUID, player.PlayingOn, player.ProtocolVersion, player.NodeId, config.GetWiredHost())
		case packet.Id_PlayerRemove:
			var player protocol.Player
			err := protocol.DecodePacket(pp.Data, &player)
			if err != nil {
				log.Println("Error decoding player remove packet:", err)
				continue
			}

			utils.RemovePlayer(player)
			log.Printf("Player %s (%s) left %s and played for %s on %s.%s\n", player.Name, player.UUID, player.PlayingOn, calculatePlaytime(player), player.NodeId, config.GetWiredHost())
		}
	}
}

func calculatePlaytime(player protocol.Player) string {
	// hours, minutes, seconds
	var playtime [3]int

	// calculate playtime
	playtime[0] = int(time.Now().Unix()-player.JoinedAt) / 3600
	playtime[1] = int(time.Now().Unix()-player.JoinedAt) / 60 % 60
	playtime[2] = int(time.Now().Unix()-player.JoinedAt) % 60

	return fmt.Sprintf("%d hours, %d minutes, %d seconds", playtime[0], playtime[1], playtime[2])
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

		clients := utils.GetClients()
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
