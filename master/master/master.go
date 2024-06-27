package master

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"time"
	"wiredmaster/routes"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/packet"
	"wired.rip/wiredutils/protocol"
)

func Run() {
	config.Init()
	go startHttpServer()
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

	http.HandleFunc("/api/connect/publickey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		w.Write(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&wiredKey.PublicKey),
		}))
	})

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
		fmt.Println("Error decoding shared secret:", err)
		return
	}

	err = conn.EnableEncryption(sharedSecret)
	if err != nil {
		fmt.Println("Error enabling AES-Encryption:", err)
		return
	}

	err = conn.SendPacket(packet.Id_Ready, nil)
	if err != nil {
		fmt.Println("Error sending ready:", err)
		return
	}

	for {
		var pp protocol.Packet
		err := pp.Read(conn)
		if err != nil {
			// fmt.Println("Error reading packet:", err)
			continue
		}

		//fmt.Println("Received packet:", pp.ID)
		//fmt.Println("Data:", string(pp.Data))

		switch pp.ID {
		case packet.Id_Hello:
			fmt.Printf("Received hello at %s\n", time.Now())
			var hello packet.Hello
			err := protocol.DecodePacket(pp.Data, &hello)
			if err != nil {
				fmt.Println("Error decoding hello:", err)
				continue
			}

			fmt.Println("Received hello:", hello)

			routes := config.GetRoutes()

			// send routes packet
			err = conn.SendPacket(packet.Id_Routes, packet.Routes{
				Routes: routes,
			})

			if err != nil {
				fmt.Println("Error sending routes:", err)
				continue
			}
		case packet.Id_Ping:
			err = conn.SendPacket(packet.Id_Pong, nil)
			if err != nil {
				fmt.Println("Error sending pong:", err)
				continue
			}

			// fmt.Println("Sent pong")
		}
	}
}
