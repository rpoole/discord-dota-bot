package main

// Make this file more object oriented.

import (
	"io/ioutil"
	"strings"
	"os/user"
	"log"
	"os"
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

func getLastMatch() string {
	if _, err := os.Stat(getHomeDir() + "/.dota-config/last_match"); err == nil {
		return fileContentsToString("/.dota-config/last_match")
	}

	return ""
}

func getHeroes() string {
	return fileContentsToString("/.dota-config/heroes.json")
}

func getApiKey() string {
	return fileContentsToString("/.dota-config/apikey.config")
}

func getDiscordToken() string {
	return fileContentsToString("/.dota-config/discord.config")
}

func setLastMatch(currentMatch string) {
	err := ioutil.WriteFile(getHomeDir() + "/.dota-config/last_match", []byte(currentMatch), 0644)
	if err != nil {
		panic(err)
	}
}
