package routes

import (
	"net/http"

	"wired.rip/wiredutils/config"
)

func SetNodeHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "hash is required"}`))
		return
	}

	arch := r.URL.Query().Get("arch")
	if arch == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "arch is required"}`))
		return
	}

	config.SetCurrentNodeHash(hash, arch)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Node hash updated"}`))
}
