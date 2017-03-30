package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"time"
	"log"
	"math"
	"os"
	"flag"
	"strings"
	"github.com/bwmarrin/discordgo"
	"github.com/mxk/go-sqlite/sqlite3"
)
//tricepzId <@111618891402182656>

const DOTA_ID string = "570"

var lastHour int = 0
var currentHour int = 0

var lastMatch string = ""

var debug bool = false

var playerDb map[string]*Player
var heroMap map[int]Hero

var db *sqlite3.Conn
var dg *discordgo.Session
var err error

type BicepzBot struct{}

type GameData struct {
	duration, radiantScore, direScore int
	radiantWin bool
}

type PlayerData struct {
	accountId, matchId, kills, deaths, assists, hero string
	win bool
}

func sendMessage(channelId string, msg string) {
	if channelId == "0" {
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
	latestMatches := make(map[string]string)

	for _, player := range playerDb {
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
		playersJson := matchesJson[0].(map[string]interface{})["players"].([]interface{})

		log.Println(fmt.Sprintf("last match - %s, current match - %s: %s", player.lastMatch, currentMatch, player.name))

		if player.lastMatch == currentMatch {
			continue
		}

		for _, v := range playersJson {
			playerAccountId := fmt.Sprintf("%.0f", v.(map[string]interface{})["account_id"].(float64))

			if _, ok := playerDb[playerAccountId]; ok && playerDb[playerAccountId].lastMatch != currentMatch {
				if currentMatch > latestMatches[playerAccountId] {
					latestMatches[playerAccountId] = currentMatch
				}
			}
		}
	}

	matches := make(map[string]map[string]bool)

	for accountId, match := range latestMatches {
		db.Exec("UPDATE players SET last_match = " + match + " WHERE account_id = " + accountId)
		playerDb[accountId].lastMatch = match

		if _, ok := matches[match]; !ok {
			matches[match] = make(map[string]bool)
		}

		matches[match][accountId] = true
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
	game.duration = int(data["result"].(map[string]interface{})["duration"].(float64))
	game.radiantScore = int(data["result"].(map[string]interface{})["radiant_score"].(float64))
	game.direScore = int(data["result"].(map[string]interface{})["dire_score"].(float64))

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

func generateDurationMsg(game GameData) string {
	hours := game.duration / 3600
	minutes := (game.duration - hours * 3600) / 60
	seconds := game.duration - (hours * 3600) - (minutes * 60)

	hoursMsg := ""
	minutesMsg := fmt.Sprintf("%d:", minutes)
	secondsMsg := fmt.Sprintf("%d", seconds)
	if hours > 0 {
		hoursMsg = fmt.Sprintf("%d:", hours)

		if minutes < 10 {
			minutesMsg = "0" + minutesMsg
		}
	}

	if seconds < 10 {
		secondsMsg = "0" + secondsMsg
	}

	matchSummary := fmt.Sprintf("Match Duration: %s%s%s\n", hoursMsg, minutesMsg, secondsMsg)

	return matchSummary
}

func getStringArrayMaxLength(array interface{}) int {
	max := 0

	switch v := array.(type) {
		default:
			fmt.Printf("Unexpected Type %T", v)
		case map[string]PlayerData:
			for _, x := range v {
				if len(playerDb[x.accountId].name + x.hero) > max {
					max = len(playerDb[x.accountId].name + x.hero)
				}
			}
		case map[string]*Player:
			for _, x := range v {
				if len(x.name) > max {
					max = len(x.name)
				}
			}
		case []string:
			for _, x := range v {
				if len(x) > max {
					max = len(x)
				}
			}
	}

	return max
}

func getPadLengthString(max int, name string) string {
	padLength := max - len(name)
	pad := ""
	for i := 0; i < padLength; i++ {
		pad += " "
	}

	return pad
}

func getStandings(report string) string {
	var standings, sql string

	if report == "day" {
		standings = "```diff\nDaily Standings Report:\n"
		sql = "SELECT name, daily_win, daily_loss, (daily_win - daily_loss) as net_win FROM players WHERE daily_win != 0 OR daily_loss != 0 ORDER BY net_win DESC, daily_win DESC, name ASC;"
	} else if report == "week" {
		standings = "```diff\nWeekly Standings Report:\n"
		sql = "SELECT name, weekly_win, weekly_loss, (weekly_win - weekly_loss) as net_win FROM players WHERE weekly_win != 0 OR weekly_loss != 0 ORDER BY net_win DESC, weekly_win DESC, name ASC;"
	}

	max := getStringArrayMaxLength(playerDb)

	for row, err := db.Query(sql); err == nil; err = row.Next() {
		var name string
		var win, loss int
		var net_win float64
		row.Scan(&name, &win, &loss, &net_win)

		sign := " "
		if win > loss {
			sign = "+"
		} else if win < loss {
			sign = "-"
		}

		standings += fmt.Sprintf("%s %s %s %d - %d  %s%d\n", sign, strings.Title(name), getPadLengthString(max, name), win, loss, sign, int(math.Abs(net_win)))
	}

	standings += "```"

	return standings
}

func resetWeeklyStandings() {
	log.Println("Resetting Weekly Standings")

	db.Exec("UPDATE players SET weekly_win = 0")
	db.Exec("UPDATE players SET weekly_loss = 0")

	updateNextWeek(db)
}

func resetDailyStandings() {
	log.Println("Resetting Daily Standings")

	db.Exec("UPDATE players SET daily_win = 0")
	db.Exec("UPDATE players SET daily_loss = 0")

	updateNextDay(db)
}

// Reads messages and sends responses
func (bb *BicepzBot) MessageParser(s *discordgo.Session, m *discordgo.MessageCreate) {
	words := string(m.Content)
	if words == "!day" {
		s.ChannelMessageSend( m.ChannelID, getStandings("day"))
	} else if words == "!week" {
		s.ChannelMessageSend( m.ChannelID, getStandings("week"))
	}
}

func main() {
	debugPtr := flag.String("debug", "normal", "Specifiy output.")
	flag.Parse()

	channelId := "0"

	if *debugPtr == "out" {
		debug = true
	}

	if *debugPtr == "beta" {
		channelId = "291354679361667072"
		debug = true
	}

	if *debugPtr == "normal" {
		channelId = "290732375539712002"

		f, err := os.OpenFile(getHomeDir() + "/dota.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

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

	defer dg.Close()

	bb := BicepzBot{}

	dg.AddHandler(bb.MessageParser)

	for {
		currentTime := time.Now()
		log.Println(currentTime)

		if getNextDay(db).Before(currentTime) {
			resetDailyStandings()
		}

		if getNextWeek(db).Before(currentTime) {
			resetWeeklyStandings()
		}

		matches := getMostRecentMatches(apiKey)

		for matchId, summaryPlayers := range matches {
			game, players := getResults(apiKey, matchId)

			if game != (GameData{}) || players != nil {
				winMsg, winPlayersMsg, winSummaryMsg, lossMsg, lossPlayersMsg, lossSummaryMsg := "", "", "", "", "", ""

				max := getStringArrayMaxLength(players)

				for _, player := range players {
					pad := getPadLengthString(max, playerDb[player.accountId].name + player.hero)

					if player.win {
						if summaryPlayers[player.accountId] {
							winMsg += strings.Title(playerDb[player.accountId].name) + ", "
							db.Exec("UPDATE players SET daily_win = daily_win + 1 WHERE account_id = " + player.accountId)
							db.Exec("UPDATE players SET weekly_win = weekly_win + 1 WHERE account_id = " + player.accountId)
						}

						winPlayersMsg += fmt.Sprintf(" > %s - %s %s%s-%s-%s\n", strings.Title(playerDb[player.accountId].name), player.hero, pad, player.kills, player.deaths, player.assists)
					} else {
						if summaryPlayers[player.accountId] {
							lossMsg += strings.Title(playerDb[player.accountId].name) + ", "
							db.Exec("UPDATE players SET daily_loss = daily_loss + 1 WHERE account_id = " + player.accountId)
							db.Exec("UPDATE players SET weekly_loss = weekly_loss + 1 WHERE account_id = " + player.accountId)
						}

						lossPlayersMsg += fmt.Sprintf(" > %s - %s %s%s-%s-%s\n", strings.Title(playerDb[player.accountId].name), player.hero, pad, player.kills, player.deaths, player.assists)
					}
				}

				if winMsg != "" {
					winMsg = strings.TrimSuffix(winMsg, ", ")
					winSummaryMsg = fmt.Sprintf("+ %s won last game:\n", winMsg)
				}

				if lossMsg != "" {
					lossMsg = strings.TrimSuffix(lossMsg, ", ")
					lossSummaryMsg = fmt.Sprintf("- %s lost last game:\n", lossMsg)
				}

				dotabuffMsg := fmt.Sprintf("<https://www.dotabuff.com/matches/%s>\n", matchId)
				opendotaMsg := fmt.Sprintf("<https://www.opendota.com/matches/%s>", matchId)

				matchSummaryMsg := generateDurationMsg(game)

				sendMessage(channelId, "```diff\n" + matchSummaryMsg + winSummaryMsg + winPlayersMsg + lossSummaryMsg + lossPlayersMsg + "```" + dotabuffMsg + opendotaMsg)
			}
		}

		time.Sleep(time.Second * 15)
	}

	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})
}
