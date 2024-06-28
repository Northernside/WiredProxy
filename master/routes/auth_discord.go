package routes

import (
	"fmt"
	"net/http"

	"wired.rip/wiredutils/config"
)

func AuthDiscord(w http.ResponseWriter, r *http.Request) {
	clientId := config.GetDiscordClientId()
	redirectUri := config.GetDiscordRedirectUri()

	http.Redirect(w, r, fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=identify", clientId, redirectUri), http.StatusSeeOther)
}
