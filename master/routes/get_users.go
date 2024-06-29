package routes

import (
	"net/http"
	"encoding/json"

	"wired.rip/wiredutils/sqlite"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	users := sqlite.GetUsers()

	marshalledUsers, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Failed to marshal users", "error": "` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(marshalledUsers)
	return
}