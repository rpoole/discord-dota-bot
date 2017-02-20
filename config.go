package main

// Make this file more object oriented.

import (
	"io/ioutil"
	"strings"
	"os/user"
	"log"
)

func fileContentsToString(configPath string) string {
	dat, err := ioutil.ReadFile(getHomeDir() + configPath)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(dat))
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal( err )
	}

	return usr.HomeDir
}

func getHeroes() string {
	return fileContentsToString("/.dota-config/heroes.json")
}

func getPlayers() string {
	return fileContentsToString("/.dota-config/players.json")
}

func getApiKey() string {
	return fileContentsToString("/.dota-config/apikey.config")
}

func getDiscordToken() string {
	return fileContentsToString("/.dota-config/discord.config")
}
