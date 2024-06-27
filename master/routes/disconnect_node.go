package routes

import (
	"log"
	"net/http"

	"wired.rip/wiredutils/packet"
	"wired.rip/wiredutils/utils"
)

func DisconnectNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// query: player_uuid, server_host

	playerUUID := r.URL.Query().Get("player_uuid")
	serverHost := r.URL.Query().Get("server_host")

	if playerUUID == "" || serverHost == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "player_uuid and server_host are required"}`))
		return
	}

	// find player
	log.Println("Disconnecting player", playerUUID, "from", serverHost)
	player := utils.FindPlayer(playerUUID, serverHost)
	if player.Name == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Player not found"}`))
		return
	}

	// send packet packet.Id_DisconnectPlayer
	conn, ok := utils.FindClient(player.NodeId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Node not found"}`))
		return
	}

	// remove player from players array
	utils.RemovePlayer(player)

	conn.SendPacket(packet.Id_DisconnectPlayer, packet.Disconnect{
		PlayerUUID: playerUUID,
		ServerHost: serverHost,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Player disconnected"}`))
}
