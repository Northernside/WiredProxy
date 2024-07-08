package routes

import (
	"net/http"

	"wired.rip/wiredutils/config"
)

func AddNode(w http.ResponseWriter, r *http.Request) {
	// query params
	nodeId := r.URL.Query().Get("node_id")
	nodePassphrase := r.URL.Query().Get("node_passphrase")

	// Create the node
	node := config.Node{
		Id:             nodeId,
		Passphrase:     nodePassphrase,
		LastConnection: 0,
	}

	// Add the node
	status := config.AddNode(node)
	if status != http.StatusOK {
		http.Error(w, "Failed to add node", status)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}
