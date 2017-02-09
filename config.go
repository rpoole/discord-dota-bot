package main

// Make this file more object oriented.

import (
	"io/ioutil"
	"strings"
	"os/user"
	"log"
	"os"
)

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal( err )
	}

	return usr.HomeDir
}

func getHeroes() string {
	dat, err := ioutil.ReadFile(getHomeDir() + "/.dota-config/heroes.json")
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(dat))
}

func getLastMatch() string {
	if _, err := os.Stat(getHomeDir() + "/.dota-config/last_match"); err == nil {
		dat, err := ioutil.ReadFile(getHomeDir() + "/.dota-config/last_match")

		if err != nil {
			panic(err)
		}
		log.Println("Last Match: " + string(dat))
		return strings.TrimSpace(string(dat))
	}

	return ""
}

func getApiKey() string {
	dat, err := ioutil.ReadFile(getHomeDir() + "/.dota-config/apikey.config")
	if err != nil {
		panic(err)
	}
	log.Println("ApiKey: " + string(dat))
	return strings.TrimSpace(string(dat))
}

func getDiscordToken() string {
	dat, err := ioutil.ReadFile(getHomeDir() + "/.dota-config/discord.config")
	if err != nil {
		panic(err)
	}
	log.Println("Discord Token: " + string(dat))
	return strings.TrimSpace(string(dat))
}

func setLastMatch(currentMatch string) {
	err := ioutil.WriteFile(getHomeDir() + "/.dota-config/last_match", []byte(currentMatch), 0644)
	if err != nil {
		panic(err)
	}
}
