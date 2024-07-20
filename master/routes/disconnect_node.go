package routes

import (
	"log"
	"net/http"

	"wired.rip/wiredutils/packet"
	"wired.rip/wiredutils/utils"
)

func DisconnectNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// query: player_uuid, proxy_host

	playerUUID := r.URL.Query().Get("player_uuid")
	proxyHost := r.URL.Query().Get("proxy_host")

	if playerUUID == "" || proxyHost == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "player_uuid and proxy_host are required"}`))
		return
	}

	// find player
	log.Printf("Disconnecting player %s from %s", playerUUID, proxyHost)
	player := utils.FindPlayer(playerUUID, proxyHost)
	if player.Name == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Player not found"}`))
		return
	}

	// send packet packet.Id_DisconnectPlayer
	conn, _, ok := utils.FindClient(player.NodeId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Node not found"}`))
		return
	}

	// remove player from players array
	utils.RemovePlayer(player)

	conn.SendPacket(packet.Id_DisconnectPlayer, packet.Disconnect{
		PlayerUUID: playerUUID,
		ProxyHost:  proxyHost,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Player disconnected"}`))
}
