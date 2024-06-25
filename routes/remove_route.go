package routes

import (
	"net/http"
	config "wiredproxy/utils/config"
)

func RemoveRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	serverHost := r.URL.Query().Get("server_host")
	serverPort := r.URL.Query().Get("server_port")
	proxyPort := r.URL.Query().Get("proxy_port")

	if serverHost == "" || serverPort == "" || proxyPort == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "server_host, server_port and proxy_port are required"}`))
		return
	}

	// remove route
	route := config.Route{
		ServerHost: serverHost,
		ServerPort: serverPort,
		ProxyPort:  proxyPort,
	}

	status := config.DeleteRoute(route)
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		w.Write([]byte(`{"message": "Route not found"}`))
		return
	}

	w.Write([]byte(`{"message": "Route removed"}`))
}
