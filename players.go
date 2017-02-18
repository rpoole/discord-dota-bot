package main

import (
	"encoding/json"
)

type Player struct {
	name, accountId, steamId string
}

func parsePlayers() map[string]Player {
	playersJson := getPlayers()

	var data map[string]interface{}

	if err := json.Unmarshal([]byte(playersJson), &data); err != nil {
		return nil
	}

	players := data["players"].([]interface{})

	playerMap := make(map[string]Player)

	for _, player := range players {
		name := player.(map[string]interface{})["name"].(string)
		accountId := player.(map[string]interface{})["account_id"].(string)
		steamId := player.(map[string]interface{})["steam_id"].(string)

		playerMap[accountId] = Player{name: name, accountId: accountId, steamId: steamId}
	}

	return playerMap
}
