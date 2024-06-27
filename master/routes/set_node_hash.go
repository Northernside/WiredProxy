package routes

import (
	"net/http"

	"wired.rip/wiredutils/config"
)

func SetNodeHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hash := r.URL.Query().Get("hash")
	config.SetCurrentNodeHash(hash)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Node hash updated"}`))
}
