package who3

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS servers (
		server_id  TEXT     PRIMARY KEY,
		last_game_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS players (
		id                INTEGER  PRIMARY KEY AUTOINCREMENT,
		steam_id          TEXT     NOT NULL,
		name              TEXT     NOT NULL,
		consecutive_count INTEGER  NOT NULL DEFAULT 0,
		server_id         TEXT     NOT NULL REFERENCES servers(server_id),
		last_updated      DATETIME NOT NULL,
		map_names         TEXT     NOT NULL DEFAULT '',
		UNIQUE(steam_id, server_id)
	);
	`
	_, err := db.Exec(ddl)
	return err
}

// ProcessGameEnd runs the full game-end logic inside a single transaction.
func ProcessGameEnd(db *sql.DB, req GameEndRequest, now time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Reset all players on this server if last game was >1 hour ago.
	if err := resetServerIfStale(tx, req.ServerID, now); err != nil {
		return fmt.Errorf("reset stale: %w", err)
	}

	// Step 2: Reset absent players (on this server, not in current game).
	activeIDs := make([]string, len(req.Players))
	for i, p := range req.Players {
		activeIDs[i] = p.SteamID
	}
	if err := resetAbsentPlayers(tx, req.ServerID, activeIDs); err != nil {
		return fmt.Errorf("reset absent: %w", err)
	}

	// Step 3: Upsert each active player.
	for _, p := range req.Players {
		if err := upsertPlayer(tx, req.ServerID, req.MapName, p.SteamID, p.Name, now); err != nil {
			return fmt.Errorf("upsert player %s: %w", p.SteamID, err)
		}
	}

	// Step 4: Upsert server timestamp.
	if err := upsertServer(tx, req.ServerID, now); err != nil {
		return fmt.Errorf("upsert server: %w", err)
	}

	return tx.Commit()
}

func resetServerIfStale(tx *sql.Tx, serverID string, now time.Time) error {
	var lastGame time.Time
	err := tx.QueryRow("SELECT last_game_at FROM servers WHERE server_id = ?", serverID).Scan(&lastGame)
	if err == sql.ErrNoRows {
		return nil // first game on this server, nothing to reset
	}
	if err != nil {
		return err
	}

	if now.Sub(lastGame) > time.Hour {
		_, err = tx.Exec(
			"UPDATE players SET consecutive_count = 0, map_names = '' WHERE server_id = ?",
			serverID,
		)
		return err
	}
	return nil
}

func resetAbsentPlayers(tx *sql.Tx, serverID string, activeSteamIDs []string) error {
	if len(activeSteamIDs) == 0 {
		// No active players — reset everyone on the server.
		_, err := tx.Exec(
			"UPDATE players SET consecutive_count = 0, map_names = '' WHERE server_id = ?",
			serverID,
		)
		return err
	}

	placeholders := make([]string, len(activeSteamIDs))
	args := make([]interface{}, 0, len(activeSteamIDs)+1)
	args = append(args, serverID)
	for i, id := range activeSteamIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		"UPDATE players SET consecutive_count = 0, map_names = '' WHERE server_id = ? AND steam_id NOT IN (%s)",
		strings.Join(placeholders, ","),
	)
	_, err := tx.Exec(query, args...)
	return err
}

func upsertPlayer(tx *sql.Tx, serverID, mapName, steamID, name string, now time.Time) error {
	// Try to fetch existing row.
	var existingCount int
	var existingMaps string
	err := tx.QueryRow(
		"SELECT consecutive_count, map_names FROM players WHERE steam_id = ? AND server_id = ?",
		steamID, serverID,
	).Scan(&existingCount, &existingMaps)

	if err == sql.ErrNoRows {
		// Insert new player.
		_, err = tx.Exec(
			`INSERT INTO players (steam_id, name, consecutive_count, server_id, last_updated, map_names)
			 VALUES (?, ?, 1, ?, ?, ?)`,
			steamID, name, serverID, now.UTC(), mapName,
		)
		return err
	}
	if err != nil {
		return err
	}

	// Update existing player: increment count, append map.
	newCount := existingCount + 1
	var newMaps string
	if existingMaps == "" {
		newMaps = mapName
	} else {
		newMaps = existingMaps + "," + mapName
	}

	_, err = tx.Exec(
		`UPDATE players SET name = ?, consecutive_count = ?, last_updated = ?, map_names = ?
		 WHERE steam_id = ? AND server_id = ?`,
		name, newCount, now.UTC(), newMaps, steamID, serverID,
	)
	return err
}

func upsertServer(tx *sql.Tx, serverID string, now time.Time) error {
	_, err := tx.Exec(
		`INSERT INTO servers (server_id, last_game_at) VALUES (?, ?)
		 ON CONFLICT(server_id) DO UPDATE SET last_game_at = excluded.last_game_at`,
		serverID, now.UTC(),
	)
	return err
}

// GetPlayersByMinCount returns players with consecutive_count >= minCount.
// If serverID is non-empty, results are filtered to that server.
func GetPlayersByMinCount(db *sql.DB, minCount int, serverID string) ([]PlayerResponse, error) {
	var rows *sql.Rows
	var err error
	if serverID != "" {
		rows, err = db.Query(
			`SELECT steam_id, name, consecutive_count, server_id, last_updated, map_names
			 FROM players WHERE consecutive_count >= ? AND server_id = ?`,
			minCount, serverID,
		)
	} else {
		rows, err = db.Query(
			`SELECT steam_id, name, consecutive_count, server_id, last_updated, map_names
			 FROM players WHERE consecutive_count >= ?`,
			minCount,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PlayerResponse
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.SteamID, &p.Name, &p.ConsecutiveCount, &p.ServerID, &p.LastUpdated, &p.MapNames); err != nil {
			return nil, err
		}
		maps := splitMaps(p.MapNames)
		results = append(results, PlayerResponse{
			SteamID:          p.SteamID,
			Name:             p.Name,
			ConsecutiveCount: p.ConsecutiveCount,
			ServerID:         p.ServerID,
			LastUpdated:      p.LastUpdated,
			MapNames:         maps,
		})
	}
	return results, rows.Err()
}

func splitMaps(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}
