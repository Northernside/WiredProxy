package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"wiredproxy/protocol"
	"wiredproxy/routes"
	"wiredproxy/utils/config"
)

func main() {
	go startProxyServer()
	startHttpServer()
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

		go handleConnection(clientConn)
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

	customHandler("/api/routes", routes.GetRoutes, http.MethodGet)
	customHandler("/api/routes/add", routes.AddRoute, http.MethodGet)
	customHandler("/api/routes/remove", routes.RemoveRoute, http.MethodDelete)

	http.ListenAndServe(":8080", nil)
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

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	var handshakePacket protocol.HandshakePacket
	err := handshakePacket.ReadFrom(clientConn)
	if err != nil {
		fmt.Println("Error reading handshake packet:", err)
		sendErrorScreen(clientConn)
		return
	}

	route, ok := config.GetRouteByProxyDomain(string(handshakePacket.Hostname))
	if !ok {
		fmt.Printf("Route not found for %s\n", handshakePacket.Hostname)
		sendErrorScreen(clientConn)
		return
	}

	handshakePacket.Hostname = protocol.String(route.ServerHost)

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", route.ServerHost, route.ServerPort))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		sendErrorScreen(clientConn)
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

func sendErrorScreen(clientConn net.Conn) {
	var statusRequest protocol.Packet

	statusRequest.ReadFrom(clientConn)

	var statusResponse protocol.StatusResponse

	newResponse := protocol.StatusResponseJSON{
		Version: protocol.Version{
			Name:     "§4Offline",
			Protocol: 0,
		},
		Players: protocol.Players{
			Max:    0,
			Online: 0,
		},
		Description: protocol.Description{
			Text: "§8[§7Wired§8] §cServer is offline",
		},
	}

	n, err := json.Marshal(newResponse)
	if err != nil {
		panic(err)
	}

	statusResponse.Status = protocol.String(n)
	statusResponse.WriteTo(clientConn)
}

func logPacket(data []byte) {
	fmt.Println(hex.Dump(data))
}
