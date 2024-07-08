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
	Online  bool   `json:"online"`
}

func GetNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var nodes []Node

	// clients is make(map[string]protocol.Conn)
	onlineNodes := utils.GetClients()
	for key, conn := range onlineNodes {
		nodes = append(nodes, Node{
			Key:     fmt.Sprintf("%s.%s", key, config.GetWiredHost()),
			Address: conn.RemoteAddr().String(),
		})
	}

	var offlineNodes []Node
	configNodes := config.GetNodes()
	// if node doesnt exist in onlineClients, add it to offlineNodes
	for _, node := range configNodes {
		if _, ok := onlineNodes[node.Id]; !ok {
			offlineNodes = append(offlineNodes, Node{
				Key:     fmt.Sprintf("%s.%s", node.Id, config.GetWiredHost()),
				Address: "",
			})
		}
	}

	w.WriteHeader(http.StatusOK)

	if len(nodes) == 0 && len(offlineNodes) == 0 {
		w.Write([]byte(`{"online": [], "offline": []}`))
		return
	}

	marshalledOnlineNodes, err := json.Marshal(nodes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Failed to marshal online nodes", "error": "` + err.Error() + `"}`))
	}

	marshalledOfflineNodes, err := json.Marshal(offlineNodes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Failed to marshal offline nodes", "error": "` + err.Error() + `"}`))
	}

	w.Write([]byte(fmt.Sprintf(`{"online": %s, "offline": %s}`, marshalledOnlineNodes, marshalledOfflineNodes)))

	return
}
