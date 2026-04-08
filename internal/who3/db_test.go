package who3

import (
	"testing"
	"time"
)

func TestIncrementConsecutiveCount(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	// Game 1: two players on server "s1" playing "campgrounds".
	req1 := GameEndRequest{
		ServerID: "s1",
		MapName:  "campgrounds",
		Players: []PlayerInfo{
			{SteamID: "steam_aaa", Name: "Alice"},
			{SteamID: "steam_bbb", Name: "Bob"},
		},
	}
	if err := ProcessGameEnd(db, req1, now); err != nil {
		t.Fatalf("game 1: %v", err)
	}

	// Both players should have count=1 and map_names=["campgrounds"].
	players, err := GetPlayersByMinCount(db, 1, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	for _, p := range players {
		if p.ConsecutiveCount != 1 {
			t.Errorf("player %s: expected count 1, got %d", p.SteamID, p.ConsecutiveCount)
		}
		if len(p.MapNames) != 1 || p.MapNames[0] != "campgrounds" {
			t.Errorf("player %s: expected [campgrounds], got %v", p.SteamID, p.MapNames)
		}
	}

	// Game 2: only Alice plays "bloodrun" 10 minutes later.
	now2 := now.Add(10 * time.Minute)
	req2 := GameEndRequest{
		ServerID: "s1",
		MapName:  "bloodrun",
		Players: []PlayerInfo{
			{SteamID: "steam_aaa", Name: "Alice"},
		},
	}
	if err := ProcessGameEnd(db, req2, now2); err != nil {
		t.Fatalf("game 2: %v", err)
	}

	// Alice should have count=2, maps=[campgrounds, bloodrun].
	// Bob should have count=0 (absent).
	players, err = GetPlayersByMinCount(db, 1, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player with count>=1, got %d", len(players))
	}
	if players[0].SteamID != "steam_aaa" {
		t.Errorf("expected steam_aaa, got %s", players[0].SteamID)
	}
	if players[0].ConsecutiveCount != 2 {
		t.Errorf("expected count 2, got %d", players[0].ConsecutiveCount)
	}
	if len(players[0].MapNames) != 2 || players[0].MapNames[0] != "campgrounds" || players[0].MapNames[1] != "bloodrun" {
		t.Errorf("expected [campgrounds bloodrun], got %v", players[0].MapNames)
	}
}

func TestAbsentPlayerReset(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	// Game 1: Alice and Bob.
	req1 := GameEndRequest{
		ServerID: "s1",
		MapName:  "aerowalk",
		Players: []PlayerInfo{
			{SteamID: "steam_aaa", Name: "Alice"},
			{SteamID: "steam_bbb", Name: "Bob"},
		},
	}
	if err := ProcessGameEnd(db, req1, now); err != nil {
		t.Fatalf("game 1: %v", err)
	}

	// Game 2: only Charlie (new player).
	now2 := now.Add(10 * time.Minute)
	req2 := GameEndRequest{
		ServerID: "s1",
		MapName:  "dm6",
		Players: []PlayerInfo{
			{SteamID: "steam_ccc", Name: "Charlie"},
		},
	}
	if err := ProcessGameEnd(db, req2, now2); err != nil {
		t.Fatalf("game 2: %v", err)
	}

	// Alice and Bob should be at 0, Charlie at 1.
	players, err := GetPlayersByMinCount(db, 1, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].SteamID != "steam_ccc" || players[0].ConsecutiveCount != 1 {
		t.Errorf("unexpected player: %+v", players[0])
	}
}

func TestOneHourServerReset(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	// Game 1: Alice plays.
	req1 := GameEndRequest{
		ServerID: "s1",
		MapName:  "toxicity",
		Players: []PlayerInfo{
			{SteamID: "steam_aaa", Name: "Alice"},
		},
	}
	if err := ProcessGameEnd(db, req1, now); err != nil {
		t.Fatalf("game 1: %v", err)
	}

	// Game 2: Alice plays again, but 2 hours later.
	now2 := now.Add(2 * time.Hour)
	req2 := GameEndRequest{
		ServerID: "s1",
		MapName:  "bloodrun",
		Players: []PlayerInfo{
			{SteamID: "steam_aaa", Name: "Alice"},
		},
	}
	if err := ProcessGameEnd(db, req2, now2); err != nil {
		t.Fatalf("game 2: %v", err)
	}

	// Even though Alice was present, the 1-hour reset should have set her count to 0
	// first, then increment to 1 (not 2).
	players, err := GetPlayersByMinCount(db, 1, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].ConsecutiveCount != 1 {
		t.Errorf("expected count 1 after 1-hour reset, got %d", players[0].ConsecutiveCount)
	}
	if len(players[0].MapNames) != 1 || players[0].MapNames[0] != "bloodrun" {
		t.Errorf("expected [bloodrun] after reset, got %v", players[0].MapNames)
	}
}

func TestDefaultMinThreshold(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	// Play 2 games — count will be 2, below default min=3.
	for i, mapName := range []string{"campgrounds", "bloodrun"} {
		req := GameEndRequest{
			ServerID: "s1",
			MapName:  mapName,
			Players:  []PlayerInfo{{SteamID: "steam_aaa", Name: "Alice"}},
		}
		if err := ProcessGameEnd(db, req, now.Add(time.Duration(i)*10*time.Minute)); err != nil {
			t.Fatalf("game %d: %v", i+1, err)
		}
	}

	players, err := GetPlayersByMinCount(db, 3, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 0 {
		t.Errorf("expected 0 players with count>=3, got %d", len(players))
	}

	// Play a 3rd game.
	req := GameEndRequest{
		ServerID: "s1",
		MapName:  "aerowalk",
		Players:  []PlayerInfo{{SteamID: "steam_aaa", Name: "Alice"}},
	}
	if err := ProcessGameEnd(db, req, now.Add(20*time.Minute)); err != nil {
		t.Fatalf("game 3: %v", err)
	}

	players, err = GetPlayersByMinCount(db, 3, "s1")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player with count>=3, got %d", len(players))
	}
	if len(players[0].MapNames) != 3 {
		t.Errorf("expected 3 maps, got %v", players[0].MapNames)
	}
}

func TestMultipleServers(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	// Alice plays on server s1.
	req1 := GameEndRequest{
		ServerID: "s1",
		MapName:  "campgrounds",
		Players:  []PlayerInfo{{SteamID: "steam_aaa", Name: "Alice"}},
	}
	if err := ProcessGameEnd(db, req1, now); err != nil {
		t.Fatalf("game s1: %v", err)
	}

	// Alice also plays on server s2.
	req2 := GameEndRequest{
		ServerID: "s2",
		MapName:  "bloodrun",
		Players:  []PlayerInfo{{SteamID: "steam_aaa", Name: "Alice"}},
	}
	if err := ProcessGameEnd(db, req2, now); err != nil {
		t.Fatalf("game s2: %v", err)
	}

	// No filter: should see 2 entries (one per server).
	players, err := GetPlayersByMinCount(db, 1, "")
	if err != nil {
		t.Fatalf("get players: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(players))
	}

	// Filter by server s1: should see 1 entry.
	players, err = GetPlayersByMinCount(db, 1, "s1")
	if err != nil {
		t.Fatalf("get players for s1: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 entry for s1, got %d", len(players))
	}
	if players[0].ServerID != "s1" {
		t.Errorf("expected server_id s1, got %s", players[0].ServerID)
	}
}
