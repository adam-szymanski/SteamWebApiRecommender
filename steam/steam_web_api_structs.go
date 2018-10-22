package steam

// GamePlaytime represents playtime of single game.
type GamePlaytime struct {
	AppID          *uint64 `json:"appid"`
	Playtime       *uint64 `json:"playtime_forever"`
	Playtime2Weeks *uint64 `json:"playtime_2weeks"`
}

// GamesOwned is set of GamePlaytime for single user.
type GamesOwned struct {
	GameCount *uint64        `json:"game_count"`
	Games     []GamePlaytime `json:"games"`
}

// GetOwnedGamesResponse is package for GamesOwned.
type GetOwnedGamesResponse struct {
	Response *GamesOwned `json:"response"`
}
