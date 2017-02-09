package main

import (
	"io/ioutil"
	"encoding/json"
	"strings"
)

type Hero struct {
	id int
	name, localizedName string
}

func getHeroes() string {
	dat, err := ioutil.ReadFile("heroes.json")
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(dat))
}

func parseHeroes() map[int]Hero {
	heroesJson := getHeroes()

	var data map[string]interface{}

	if err := json.Unmarshal([]byte(heroesJson), &data); err != nil {
		return nil
	}

	heroes := data["heroes"].([]interface{})

	heroMap := make(map[int]Hero)

	for _, hero := range heroes {
		id := int(hero.(map[string]interface{})["id"].(float64))
		name := hero.(map[string]interface{})["name"].(string)
		localizedName := hero.(map[string]interface{})["localized_name"].(string)

		heroMap[id] = Hero{id: id, name: name, localizedName: localizedName}
	}

	return heroMap
}
