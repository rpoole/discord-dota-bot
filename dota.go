package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"time"
	"log"
	"os"
	"strings"
	"github.com/bwmarrin/discordgo"
	"github.com/mxk/go-sqlite/sqlite3"
)
//tricepzId <@111618891402182656>

var lastMatch string = ""
var tricepzId string

var debug bool = false

var heroMap map[int]Hero
var playerMap map[string]Player

var db *sqlite3.Conn
var dg *discordgo.Session
var err error

type GameData struct {
	duration, radiantScore, direScore float64
	radiantWin bool
}

type PlayerData struct {
	accountId, matchId, kills, deaths, assists, hero string
	win bool
}

func sendMessage(channelId string, msg string) {
	if debug {
		log.Println(msg)
	} else {
		dg.ChannelMessageSend(channelId, msg)
	}
}

func makeRequest(url string) map[string]interface{} {
	response, err := http.Get(url)

	if err != nil {
		return nil
	}

	raw, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil
	}

	var data map[string]interface{}

	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}

	return data
}

func getResults(apiKey string) (GameData, map[string]PlayerData) {
	matchHistoryUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchHistory/V001/?account_id=" + tricepzId + "&key=" + apiKey

	log.Println(matchHistoryUrl)

	data := makeRequest(matchHistoryUrl)

	var matches []interface{}
	if data != nil && data["result"] != nil {
		matches = data["result"].(map[string]interface{})["matches"].([]interface{})
	} else {
		fmt.Println(data)
		return GameData{}, nil
	}

	log.Printf("Got current match id: %.0f\n", matches[0].(map[string]interface{})["match_id"].(float64))

	currentMatch := fmt.Sprintf("%.0f", matches[0].(map[string]interface{})["match_id"].(float64))
	if currentMatch == lastMatch {
		log.Printf("Current match is the same as last: %s - %s\n", currentMatch, lastMatch)
		return GameData{}, nil
	}

	db.Exec("UPDATE players SET last_match = " + currentMatch + " WHERE account_id = " + tricepzId)
	lastMatch = currentMatch

	matchDetailsUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchDetails/V001/?match_id=" + lastMatch + "&key=" + apiKey
	log.Println(matchDetailsUrl)

	data = makeRequest(matchDetailsUrl)

	var playerDetails []interface{}
	if data != nil && data["result"] != nil {
		playerDetails = data["result"].(map[string]interface{})["players"].([]interface{})
	} else {
		return GameData{}, nil
	}

	var game GameData

	game.radiantWin = data["result"].(map[string]interface{})["radiant_win"].(bool)
	game.duration = data["result"].(map[string]interface{})["duration"].(float64)
	game.radiantScore = data["result"].(map[string]interface{})["radiant_score"].(float64)
	game.direScore = data["result"].(map[string]interface{})["dire_score"].(float64)

	friends := make(map[string]PlayerData);

	// Print match IDs
	for _, v := range playerDetails {

		accountId := fmt.Sprintf("%.0f", v.(map[string]interface{})["account_id"].(float64))
		if _, ok := playerMap[accountId]; ok {
			// Set player data
			stats := PlayerData{matchId: currentMatch}
			stats.accountId = accountId
			stats.kills = fmt.Sprintf("%.0f", v.(map[string]interface{})["kills"].(float64))
			stats.deaths = fmt.Sprintf("%.0f", v.(map[string]interface{})["deaths"].(float64))
			stats.assists = fmt.Sprintf("%.0f", v.(map[string]interface{})["assists"].(float64))
			stats.hero = heroMap[int(v.(map[string]interface{})["hero_id"].(float64))].localizedName
			playerSlot := v.(map[string]interface{})["player_slot"].(float64)

			log.Println(stats)

			// Determine if player won.
			stats.win = false
			if game.radiantWin == true && playerSlot < 5 || game.radiantWin == false && playerSlot > 100 {
				stats.win = true
			}

			friends[accountId] = stats
		}
	}

	return game, friends
}


func main() {

	// Check for debug flag.
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 && argsWithoutProg[0] == "debug" {
		debug = true;
	}

	// Get config info.
	token := getDiscordToken()
	apiKey := getApiKey()

	// Get our hero and players map for easy lookups.
	heroMap = parseHeroes()
	playerMap = parsePlayers()

	db, _ = sqlite3.Open(getHomeDir() + "/.dota-config/dota.db")

	// Find tricepz ID.
	sql := "SELECT account_id, * FROM players WHERE name = 'zack'"
	for row, err := db.Query(sql); err == nil; err = row.Next() {
		row.Scan(&tricepzId)
		log.Println(tricepzId)
	}

	// Get last match.
	if !debug {
		sql := "SELECT last_match, * FROM players WHERE name = 'zack'"
		for row, err := db.Query(sql); err == nil; err = row.Next() {
			row.Scan(&lastMatch)
			log.Println(lastMatch)
		}
	}

	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Println(err)
		return
	}

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("Error opening Discord session: ", err)
	}

	// generalId := "111616828899381248"
	codingId := "151877780605239296"

	for {
		game, players := getResults(apiKey)
		if game != (GameData{}) || players != nil {
			// Set win loss string.
			var result string
			result = "lost"
			if players[tricepzId].win == true {
				result = "won"
			}

			var teamMemberMsg string = ""
			for _, player := range players {
				if player.win == players[tricepzId].win && player.accountId != tricepzId {
					teamMemberMsg += fmt.Sprintf(" - %s as %s with K/D/A: %s/%s/%s\n", strings.Title(playerMap[player.accountId].name), player.hero, player.kills, player.deaths, player.assists)
				}
			}

			summaryMsg := fmt.Sprintf("Tricepz %s his last game as %s with K/D/A: %s/%s/%s", result, players[tricepzId].hero, players[tricepzId].kills, players[tricepzId].deaths, players[tricepzId].assists)
			dotabuffMsg := fmt.Sprintf("https://www.dotabuff.com/matches/%s", players[tricepzId].matchId)
			opendotaMsg := fmt.Sprintf("https://www.opendota.com/matches/%s", players[tricepzId].matchId)

			sendMessage(codingId, summaryMsg + "\n" + teamMemberMsg + dotabuffMsg + "\n" + opendotaMsg)

		}
		time.Sleep(time.Second * 10)
	}

	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})
}
