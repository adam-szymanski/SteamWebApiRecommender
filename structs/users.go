package structs

import (
	"SteamWebApiCrawler/bytechunk"
	fmt "fmt"
	"os"

	"github.com/golang/protobuf/proto"
)

// Users contains and manages prousers.
type Users struct {
	Values map[uint64]*User
}

// CreateUsers returns new map of users.
func CreateUsers() *Users {
	return &Users{
		Values: make(map[uint64]*User),
	}
}

// GetUser returns user with given ID. User is created if needed.
func (users *Users) GetUser(userID uint64) *User {
	user, ok := users.Values[userID]
	if !ok {
		user = &User{
			UserId:        userID,
			GamesPlaytime: make(map[uint64]*GamePlaytime),
		}
		users.Values[userID] = user
	}
	return user
}

// WriteUsers saves users in binary file.
func (users *Users) WriteUsers(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, user := range users.Values {
		if len(user.GamesPlaytime) == 0 {
			continue
		}
		bytes, err := proto.Marshal(user)
		if err != nil {
			fmt.Println(err)
			continue
		}
		bytechunk.WriteBytes(f, bytes)
	}
}

// LoadUsers reads binary written users from given file. Users owning less than minGamesOwned are omitted.
func (users *Users) LoadUsers(filename string, minGamesOwned uint32) {
	fmt.Println("Loading users.")
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("Could not read users file.")
		return
	}
	defer f.Close()

	var usersLoaded uint32
	var gamesOwned uint32
	for reader := bytechunk.CreateByteReader(f); !reader.ReadNext(); {
		if err != nil {
			panic(err)
		}
		user := &User{}
		err := proto.Unmarshal(reader.GetValue(), user)
		if err != nil {
			fmt.Println("Error when unmarshaling User: ", reader.GetValue())
			continue
		}
		gamesOwned += uint32(len(user.GamesPlaytime))
		usersLoaded++
		if usersLoaded%100000 == 0 {
			fmt.Println("Loaded ", usersLoaded, " users owning ", gamesOwned, " games.")
		}
		if minGamesOwned > uint32(len(user.GamesPlaytime)) {
			continue
		}
		users.Values[user.UserId] = user
	}
	fmt.Println("User loading finished. Loaded ", usersLoaded, " users owning ", gamesOwned, " games.")
}
