package utils

import (
	"sync"

	"wired.rip/wiredutils/protocol"
)

var (
	PlayersArray = make([]protocol.Player, 0)
	PlayersMux   = &sync.Mutex{}
)

func AddPlayer(player protocol.Player) {
	PlayersMux.Lock()
	defer PlayersMux.Unlock()

	PlayersArray = append(PlayersArray, player)
}

func RemovePlayer(player protocol.Player) {
	PlayersMux.Lock()
	defer PlayersMux.Unlock()

	for i, p := range PlayersArray {
		p.Conn = nil
		if p == player {
			PlayersArray = append(PlayersArray[:i], PlayersArray[i+1:]...)
			break
		}
	}
}

func FindPlayer(uuid string, host string) protocol.Player {
	PlayersMux.Lock()
	defer PlayersMux.Unlock()

	for _, player := range PlayersArray {
		if player.UUID == uuid && player.PlayingOn == host {
			return player
		}
	}

	return protocol.Player{}
}
