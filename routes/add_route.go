package routes

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	config "wiredproxy/utils/config"
)

func AddRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	serverHost := r.URL.Query().Get("server_host")
	serverPort := r.URL.Query().Get("server_port")
	proxyDomain := r.URL.Query().Get("proxy_domain")
	proxyPort := r.URL.Query().Get("proxy_port")

	if serverHost == "" || serverPort == "" || proxyDomain == "" || proxyPort == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "server_host, server_port, proxy_domain and proxy_port are required"}`))
		return
	}

	// add route
	route := config.Route{
		RouteId:     randomId(),
		ServerHost:  serverHost,
		ServerPort:  serverPort,
		ProxyDomain: proxyDomain,
		ProxyPort:   proxyPort,
	}

	status := config.AddRoute(route)
	w.WriteHeader(status)
	if status == http.StatusConflict {
		w.Write([]byte(`{"message": "Proxy port already in use"}`))
		return
	}

	w.Write([]byte(`{"message": "Route added", "route_id": "` + route.RouteId + `"}`))
}

func randomId() string {
	b := make([]byte, 4)

	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(b)
}
