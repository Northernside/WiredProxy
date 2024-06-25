package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"wiredproxy/routes"
	handshake "wiredproxy/utils"
	"wiredproxy/utils/config"
)

func main() {
	go loadRoutes()
	startHttpServer()
}

func loadRoutes() {
	routes := config.GetRoutes()
	for _, route := range routes {
		go startProxyServer(route)
	}
}

func startProxyServer(route config.Route) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", route.ProxyPort))
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Routing %s:%s on %s:%s\n", route.ServerHost, route.ServerPort, "127.0.0.1", route.ProxyPort)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}

		go handleConnection(clientConn, route)
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

func handleConnection(clientConn net.Conn, route config.Route) {
	defer clientConn.Close()

	handshakePacket, err := handshake.ReadPacket(clientConn)
	if err != nil {
		fmt.Println("Error reading handshake packet:", err)
		return
	}

	editedHandshakePacket := *handshakePacket
	editedHandshakePacket.Hostname = handshake.String(route.ServerHost)

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", route.ServerHost, route.ServerPort))
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
