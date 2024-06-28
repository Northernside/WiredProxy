package routes

import (
	"net/http"

	"wired.rip/wiredutils/sqlite"
)

func ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	discordId := r.FormValue("discord_id")
	role := r.FormValue("role")

	if discordId == "" || role == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Missing discord_id or role"}`))
		return
	}

	err := sqlite.ChangeUserRole(discordId, role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Failed to change user role", "error": "` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "User role changed"}`))
}
