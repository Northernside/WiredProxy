package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/utils"
)

type Node struct {
	Key     string `json:"key"`
	Address string `json:"address"`
}

func GetNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var nodes []Node

	// clients is make(map[string]protocol.Conn)
	clients := utils.GetClients()
	for key, conn := range clients {
		nodes = append(nodes, Node{
			Key:     fmt.Sprintf("%s.%s", key, config.GetWiredHost()),
			Address: conn.RemoteAddr().String(),
		})
	}

	w.WriteHeader(http.StatusOK)

	if len(nodes) == 0 {
		w.Write([]byte(`[]`))
		return
	}

	marshalledNodes, err := json.Marshal(nodes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Failed to marshal nodes", "error": "` + err.Error() + `"}`))
		return
	}

	w.Write(marshalledNodes)
	return
}
