package main

import (
	"SteamWebApiCrawler/bytechunk"
	"SteamWebApiCrawler/steam"
	"SteamWebApiCrawler/structs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
)

func updateUsers(file *os.File, users *structs.Users) {
	var responsesNum int
	var usersUpdated int
	var gameOwned uint64
	var newUsers int
	for reader := bytechunk.CreateByteReader(file); !reader.ReadNext(); {
		var response structs.UserHttpResponse
		err := proto.Unmarshal(reader.GetValue(), &response)
		if err != nil {
			fmt.Println("Error when unmarshaling response ", responsesNum, ": ", string(reader.GetValue()))
			continue
		}
		responsesNum++
		if response.GetHttpResponse().GetResponseCode() != 200 {
			continue
		}

		ownedGames := steam.GetOwnedGamesResponse{}
		err = json.Unmarshal(response.GetHttpResponse().Body, &ownedGames)
		if err != nil {
			fmt.Println(string(response.GetHttpResponse().Body))
			panic(err)
		}
		if ownedGames.Response == nil || ownedGames.Response.GameCount == nil {
			continue
		}

		gameOwned += *ownedGames.Response.GameCount
		userID := response.GetUserId()
		oldLen := len(users.Values)
		user := users.GetUser(userID)
		newUsers += len(users.Values) - oldLen
		gamesPlaytime := user.GamesPlaytime

		if user.GetLastOwnedGamesUpdate() < response.GetHttpResponse().GetTimestamp() {
			for _, game := range ownedGames.Response.Games {
				playTime, ok := gamesPlaytime[*game.AppID]
				if !ok {
					playTime = &structs.GamePlaytime{}
					gamesPlaytime[*game.AppID] = playTime
				}
				if game.Playtime != nil {
					playTime.Playtime = *game.Playtime
				}
				if game.Playtime2Weeks != nil {
					playTime.Playtime_2Weeks = *game.Playtime2Weeks
				}
			}
			usersUpdated++
		}
		if responsesNum%10000 == 0 {
			fmt.Println("Parsed ", responsesNum, " responses and updates ", usersUpdated, " users owning ", gameOwned, " games. newUsers: ", newUsers)
		}
	}
	fmt.Println("Finished parsing. Parsed ", responsesNum, " responses and updates ", usersUpdated, " users owning ", gameOwned, " games. newUsers: ", newUsers)
}

func main() {
	/*f, err := os.Create("cpu_profile.txt")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()*/
	users := structs.CreateUsers()
	users.LoadUsers("../data/users.binary", 1)

	os.Mkdir("../data", 0777)
	files, err := ioutil.ReadDir("../dump")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		fmt.Println(file.Name())
		if file.IsDir() {
			continue
		}
		if strings.Index(file.Name(), "get_owned_games") == 0 {
			file, err := os.Open("../dump/" + file.Name())
			if err != nil {
				fmt.Println("Could not open ", "../dump/"+file.Name(), " because: ", err)
				continue
			}
			updateUsers(file, users)
			defer file.Close()
		}
	}
	users.WriteUsers("../data/users.binary")
}
