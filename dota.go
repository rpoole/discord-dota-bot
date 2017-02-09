package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"time"
	"log"
	"strings"
	"os"
)

var lastMatch string = ""
var homeDir string = ""
var debug bool = false
var heroMap map[int]Hero

type GameData struct {
	matchId, kills, deaths, assists, hero string
	win bool
}

func sendMessage(dg *discordgo.Session, channelId string, msg string) {
	if debug {
		log.Println(msg)
	} else {
		dg.ChannelMessageSend(channelId, msg)
	}
}

func getLastMatch() string {
	if _, err := os.Stat(homeDir + "/.dota-config/last_match"); err == nil {
		dat, err := ioutil.ReadFile(homeDir + "/.dota-config/last_match")

		if err != nil {
			panic(err)
		}
		log.Println("Last Match: " + string(dat))
		return strings.TrimSpace(string(dat))
	}

	return ""
}

func getApiKey() string {
	dat, err := ioutil.ReadFile(homeDir + "/.dota-config/apikey.config")
	if err != nil {
		panic(err)
	}
	log.Println("ApiKey: " + string(dat))
	return strings.TrimSpace(string(dat))
}

func getDiscordToken() string {
	dat, err := ioutil.ReadFile(homeDir + "/.dota-config/discord.config")
	if err != nil {
		panic(err)
	}
	log.Println("Discord Token: " + string(dat))
	return strings.TrimSpace(string(dat))
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

func getResults(apiKey string) *GameData {
	tricepzId := "83633790"

	matchHistoryUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchHistory/V001/?account_id=" + tricepzId + "&key=" + apiKey

	log.Println(matchHistoryUrl)

	data := makeRequest(matchHistoryUrl)

	var matches []interface{}
	if data != nil && data["result"] != nil {
		matches = data["result"].(map[string]interface{})["matches"].([]interface{})
	} else {
		fmt.Println(data)
		return nil
	}

	// Print match IDs
	// for _, v := range matches {
	// 	fmt.Printf("%.0f\n", v.(map[string]interface{})["match_id"])
	// }
	// fmt.Printf("%.0f\n", matches[0].(map[string]interface{})["match_id"])

	log.Printf("Got current match id: %.0f\n", matches[0].(map[string]interface{})["match_id"].(float64))

	currentMatch := fmt.Sprintf("%.0f", matches[0].(map[string]interface{})["match_id"].(float64))
	if currentMatch == lastMatch {
		log.Printf("Current match is the same as last: %s - %s\n", currentMatch, lastMatch)
		return nil
	}

	err := ioutil.WriteFile(homeDir + "/.dota-config/last_match", []byte(currentMatch), 0644)
	if err != nil {
		panic(err)
	}

	lastMatch = currentMatch

	matchDetailsUrl := "https://api.steampowered.com/IDOTA2Match_570/GetMatchDetails/V001/?match_id=" + lastMatch + "&key=" + apiKey
	log.Println(matchDetailsUrl)

	data = makeRequest(matchDetailsUrl)

	var playerDetails []interface{}
	if data != nil && data["result"] != nil {
		playerDetails = data["result"].(map[string]interface{})["players"].([]interface{})
	} else {
		return nil
	}

	radiantWin := data["result"].(map[string]interface{})["radiant_win"].(bool)

	stats := GameData{matchId: currentMatch}

	// Print match IDs
	for _, v := range playerDetails {

		accountId := fmt.Sprintf("%.0f", v.(map[string]interface{})["account_id"].(float64))
		if accountId == tricepzId {

			stats.kills = fmt.Sprintf("%.0f", v.(map[string]interface{})["kills"].(float64))
			stats.deaths = fmt.Sprintf("%.0f", v.(map[string]interface{})["deaths"].(float64))
			stats.assists = fmt.Sprintf("%.0f", v.(map[string]interface{})["assists"].(float64))
			stats.hero = heroMap[int(v.(map[string]interface{})["hero_id"].(float64))].localizedName
			playerSlot := v.(map[string]interface{})["player_slot"].(float64)

			log.Println(stats)

			if radiantWin == true && playerSlot < 5 || radiantWin == false && playerSlot > 100 {
				stats.win = true
			} else {
				stats.win = false
			}

			break
		}
	}

	return &stats
}

func main() {

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 && argsWithoutProg[0] == "debug" {
		debug = true;
	}

	heroMap = parseHeroes()

	usr, err := user.Current()
	if err != nil {
		log.Fatal( err )
	}
	homeDir = usr.HomeDir

	if !debug {
		lastMatch = getLastMatch()
	}

	token := getDiscordToken()

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Register messageCreate as a callback for the messageCreate events.
	// dg.AddHandler(messageCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// generalId := "111616828899381248"
	codingId := "151877780605239296"
	apiKey := getApiKey()

	for {
		stats := getResults(apiKey)
		if stats != nil {
			var result string
			if stats.win == true {
				result = "won"
			} else {
				result = "lost"
			}

			//tricepzId <@111618891402182656>

			summaryMsg := fmt.Sprintf("Tricepz %s his last game on %s with K/D/A: %s/%s/%s", result, stats.hero, stats.kills, stats.deaths, stats.assists)
			dotabuffMsg := fmt.Sprintf("https://www.dotabuff.com/matches/%s", stats.matchId)
			opendotaMsg := fmt.Sprintf("https://www.opendota.com/matches/%s", stats.matchId)

			sendMessage(dg, codingId, summaryMsg + "\n" + dotabuffMsg + "\n" + opendotaMsg)

		}
		time.Sleep(time.Second * 10)
	}

	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})
}
