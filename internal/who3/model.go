package who3

import "time"

type PlayerInfo struct {
	SteamID string `json:"steam_id"`
	Name    string `json:"name"`
}

type GameEndRequest struct {
	ServerID string       `json:"server_id"`
	MapName  string       `json:"map_name"`
	Players  []PlayerInfo `json:"players"`
}

type Player struct {
	ID               int       `json:"id"`
	SteamID          string    `json:"steam_id"`
	Name             string    `json:"name"`
	ConsecutiveCount int       `json:"consecutive_count"`
	ServerID         string    `json:"server_id"`
	LastUpdated      time.Time `json:"last_updated"`
	MapNames         string    `json:"-"`
}

type PlayerResponse struct {
	SteamID          string   `json:"steam_id"`
	Name             string   `json:"name"`
	ConsecutiveCount int      `json:"consecutive_count"`
	ServerID         string   `json:"server_id"`
	MapNames         []string `json:"map_names"`
}
