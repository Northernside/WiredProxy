package routes

import (
	"net/http"
	config "wiredproxy/utils/config"
)

func RemoveRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	routeId := r.URL.Query().Get("route_id")

	if routeId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "route_id is required"}`))
		return
	}

	status := config.DeleteRoute(routeId)
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		w.Write([]byte(`{"message": "Route not found"}`))
		return
	}

	w.Write([]byte(`{"message": "Route removed"}`))
}
