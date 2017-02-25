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

const DOTA_ID string = "570"

var lastMatch string = ""

var debug bool = false

var playerDb map[string]*Player
var heroMap map[int]Hero
var playerMap map[string]*Player
var skipMap map[string]bool

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

func isPlayingDota2(apiKey string, player *Player) bool {
	steamStatusUrl := "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?steamids=" + player.steamId + "&key=" + apiKey

	data := makeRequest(steamStatusUrl)

	var players []interface{}
	if data != nil && data["response"] != nil {
		players = data["response"].(map[string]interface{})["players"].([]interface{})

		if _, ok := players[0].(map[string]interface{})["game_id"]; ok {
			if players[0].(map[string]interface{})["game_id"].(string) == DOTA_ID {
				return true
			}
		}
	}

	return false;
}

func getResults(apiKey string, player *Player) (GameData, map[string]PlayerData) {
	matchHistoryUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchHistory/V001/?account_id=" + player.accountId + "&key=" + apiKey

	log.Println(matchHistoryUrl)

	data := makeRequest(matchHistoryUrl)

	var matches []interface{}
	if data != nil && data["result"] != nil {
		matches = data["result"].(map[string]interface{})["matches"].([]interface{})
	} else {
		fmt.Println(data)
		return GameData{}, nil
	}

	log.Println(player)
	log.Println("Player Last Match: " + player.lastMatch)
	log.Printf("Got current match id: %.0f\n", matches[0].(map[string]interface{})["match_id"].(float64))

	currentMatch := fmt.Sprintf("%.0f", matches[0].(map[string]interface{})["match_id"].(float64))
	if player.lastMatch == currentMatch {
		log.Printf("Current match is the same as last: %s - %s\n", currentMatch, player.lastMatch)
		return GameData{}, nil
	}

	matchDetailsUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchDetails/V001/?match_id=" + currentMatch + "&key=" + apiKey
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

			// Make sure we skip all already parsed players
			skipMap[accountId] = true
		}
	}

	updateLastMatch(currentMatch)

	return game, friends
}

func updateLastMatch(currentMatch string) {
	for _, player := range playerDb {
		log.Println(player)
	}

	for accountId, _ := range skipMap {
		if _, ok := playerDb[accountId]; ok {
			db.Exec("UPDATE players SET last_match = " + currentMatch + " WHERE account_id = " + accountId)
			playerDb[accountId].lastMatch = currentMatch
			fmt.Println("Updating " + playerMap[accountId].name)
		}
	}

	skipMap = make(map[string]bool)
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

	playerDb = make(map[string]*Player)
	skipMap = make(map[string]bool)

	db, _ = sqlite3.Open(getHomeDir() + "/.dota-config/dota.db")

	// Get player info.
	sql := "SELECT name, account_id, last_match FROM players"
	for row, err := db.Query(sql); err == nil; err = row.Next() {

		var name, accountId, lastMatch string

		row.Scan(&name, &accountId, &lastMatch)

		if debug {
			lastMatch = ""
		}

		playerDb[accountId] = &Player{name: name, accountId: accountId, lastMatch: lastMatch}
		log.Println(playerDb[accountId])
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
		for accountId, _ := range playerDb {
			game, players := getResults(apiKey, playerDb[accountId])

			if game != (GameData{}) || players != nil {
				// Set win loss string.
				var result string
				result = "lost"
				if players[accountId].win == true {
					result = "won"
				}

				var playerListMsg string = ""
				var teamMemberMsg string = ""
				for _, player := range players {
					if player.win == players[accountId].win {
						playerListMsg += strings.Title(playerMap[player.accountId].name) + ", "
						teamMemberMsg += fmt.Sprintf(" - %s as %s with K/D/A: %s/%s/%s\n", strings.Title(playerMap[player.accountId].name), player.hero, player.kills, player.deaths, player.assists)
					}
				}

				playerListMsg = strings.TrimSuffix(playerListMsg, ", ")

				summaryMsg := fmt.Sprintf("%s %s last game:", playerListMsg, result)
				dotabuffMsg := fmt.Sprintf("<https://www.dotabuff.com/matches/%s>", players[accountId].matchId)
				opendotaMsg := fmt.Sprintf("<https://www.opendota.com/matches/%s>", players[accountId].matchId)

				sendMessage(codingId, summaryMsg + "\n" + teamMemberMsg + dotabuffMsg + "\n" + opendotaMsg)

			}
		}
		// Clear map
		skipMap = make(map[string]bool)

		time.Sleep(time.Second * 10)
	}

	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})
}
