package routes

import (
	"encoding/json"
	"net/http"

	"wired.rip/wiredutils/config"
)

func GetRoutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	routes := config.GetRoutes()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"routes": routes,
	})
}
