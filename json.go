package main

import (
	"encoding/json"
	"net/http"
	"fmt"
	"bytes"
	"io/ioutil"
	"strings"
)

var strawPollUrl string = "https://strawpoll.me/api/v2/polls"

type GetResponsePoll struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Multi bool `json:"multi"`
	Options []string `json:"options"`
	Votes []int `json:"votes"`
}

type PostRequestPoll struct {
	Title string `json:"title"`
	Options []string `json:"options"`
	Multi bool `json:"multi"`
}

type PostResponsePoll struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Options []string `json:"options"`
	Multi bool `json:"multi"`
	Dupcheck string `json:"dupcheck"`
	Captcha bool `json:"captcha"`
}

func makeStrawPoll(players map[string]PlayerData, playerDb map[string]*Player, win bool) int {
	stuff := PostRequestPoll{}

	stuff.Multi = true

	if win {
		stuff.Title = "Who was the MVP?"
	} else {
		stuff.Title = "Who threw the game?"
	}

	for _, player := range players {
		if player.win == win {
			stuff.Options = append(stuff.Options, strings.Title(playerDb[player.accountId].name))
		}
	}

	test, _ := json.Marshal(stuff)
	fmt.Println(string(test))

	response, err := http.Post(strawPollUrl, "application/json", bytes.NewBuffer(test))

	fmt.Println(response)

	if err != nil {
		fmt.Println(err)
		return -1
	}

	raw, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		fmt.Println(err)
		return -1
	}

	res := PostResponsePoll{}
    json.Unmarshal([]byte(raw), &res)

	return res.ID
}
