package routes

import (
	"net/http"

	"wired.rip/wiredutils/config"
)

func DeleteNode(w http.ResponseWriter, r *http.Request) {
	// query params
	nodeId := r.URL.Query().Get("node_id")

	// Delete the node
	status := config.DeleteNode(nodeId)
	if status != http.StatusOK {
		http.Error(w, "Failed to delete node", status)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}
