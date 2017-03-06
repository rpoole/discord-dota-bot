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

func getMostRecentMatches(apiKey string) map[string]map[string]bool {
	matches := make(map[string]map[string]bool)

	for accountId, player := range playerDb {
		matchHistoryUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchHistory/V001/?account_id=" + player.accountId + "&key=" + apiKey

		data := makeRequest(matchHistoryUrl)

		if data == nil || data["result"] == nil {
			continue
		}

		resultsJson := data["result"].(map[string]interface{})

		if resultsJson["status"].(float64) == 15 {
			log.Println(strings.Title(player.name) + " has no exposed api stats.")
			continue
		}

		matchesJson := resultsJson["matches"].([]interface{})

		currentMatch := fmt.Sprintf("%.0f", matchesJson[0].(map[string]interface{})["match_id"].(float64))

		log.Println(fmt.Sprintf("last match - %s, current match - %s: %s", player.lastMatch, currentMatch, player.name))

		if player.lastMatch == currentMatch {
			continue
		}

		db.Exec("UPDATE players SET last_match = " + currentMatch + " WHERE account_id = " + accountId)
		player.lastMatch = currentMatch

		if _, ok := matches[currentMatch]; !ok {
			matches[currentMatch] = make(map[string]bool)
		}

		matches[currentMatch][player.accountId] = true
	}

	return matches
}

func getResults(apiKey string, currentMatch string) (GameData, map[string]PlayerData) {

	matchDetailsUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchDetails/V001/?match_id=" + currentMatch + "&key=" + apiKey

	data := makeRequest(matchDetailsUrl)

	if data == nil || data["result"] == nil {
		return GameData{}, nil
	}

	playerDetails := data["result"].(map[string]interface{})["players"].([]interface{})

	var game GameData

	game.radiantWin = data["result"].(map[string]interface{})["radiant_win"].(bool)
	game.duration = data["result"].(map[string]interface{})["duration"].(float64)
	game.radiantScore = data["result"].(map[string]interface{})["radiant_score"].(float64)
	game.direScore = data["result"].(map[string]interface{})["dire_score"].(float64)

	friends := make(map[string]PlayerData);

	// Print match IDs
	for _, v := range playerDetails {

		accountId := fmt.Sprintf("%.0f", v.(map[string]interface{})["account_id"].(float64))
		if _, ok := playerDb[accountId]; ok {
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

	f, err := os.OpenFile(getHomeDir() + "/dota.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.SetOutput(f)

	// Get config info.
	token := getDiscordToken()
	apiKey := getApiKey()

	// Get list of heroes for easy lookups.
	heroMap = parseHeroes()

	// Get players.
	playerDb = make(map[string]*Player)

	db, _ = sqlite3.Open(getHomeDir() + "/.dota-config/dota.db")

	// Get player info.
	sql := "SELECT name, account_id, last_match FROM players"
	for row, err := db.Query(sql); err == nil; err = row.Next() {
		player := new(Player)
		row.Scan(&player.name, &player.accountId, &player.lastMatch)

		if debug {
			player.lastMatch = ""
		}

		playerDb[player.accountId] = player
		log.Println(playerDb[player.accountId])
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
		matches := getMostRecentMatches(apiKey)

		for matchId, summaryPlayers := range matches {
			game, players := getResults(apiKey, matchId)

			if game != (GameData{}) || players != nil {
				winMsg, winPlayersMsg, winSummaryMsg, lossMsg, lossPlayersMsg, lossSummaryMsg := "", "", "", "", "", ""

				for _, player := range players {
					if player.win {
						if summaryPlayers[player.accountId] {
							winMsg += strings.Title(playerDb[player.accountId].name) + ", "
						}

						winPlayersMsg += fmt.Sprintf(" - %s as %s with K/D/A: %s/%s/%s\n", strings.Title(playerDb[player.accountId].name), player.hero, player.kills, player.deaths, player.assists)
					} else {
						if summaryPlayers[player.accountId] {
							lossMsg += strings.Title(playerDb[player.accountId].name) + ", "
						}

						lossPlayersMsg += fmt.Sprintf(" - %s as %s with K/D/A: %s/%s/%s\n", strings.Title(playerDb[player.accountId].name), player.hero, player.kills, player.deaths, player.assists)
					}
				}

				if winMsg != "" {
					winMsg = strings.TrimSuffix(winMsg, ", ")
					winSummaryMsg = fmt.Sprintf("%s won last game:\n", winMsg)
				}

				if lossMsg != "" {
					lossMsg = strings.TrimSuffix(lossMsg, ", ")
					lossSummaryMsg = fmt.Sprintf("%s lost last game:\n", lossMsg)
				}

				dotabuffMsg := fmt.Sprintf("<https://www.dotabuff.com/matches/%s>\n", matchId)
				opendotaMsg := fmt.Sprintf("<https://www.opendota.com/matches/%s>", matchId)

				sendMessage(codingId, winSummaryMsg + winPlayersMsg + lossSummaryMsg + lossPlayersMsg + dotabuffMsg + opendotaMsg)
			}
		}

		time.Sleep(time.Second * 15)
	}

	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})
}
