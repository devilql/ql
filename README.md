# ql

Bunch of miscellaneous QuakeLive related functions and services.

## who3 — Consecutive Game Tracker

A REST service that tracks how many consecutive games each player has played on a QuakeLive server. When a game ends, the server POSTs the results. Players who keep playing see their streak count go up; players who leave get reset.

### Features

- Tracks consecutive game count per player per server
- Records the list of maps played during a streak
- Automatically resets all players on a server if the last game was over 1 hour ago
- Query players who have played N or more consecutive games (default 3)

### Running

```sh
go build ./cmd/who3
./who3 -port :8081 -db who3.db
```

| Flag    | Default    | Description              |
|---------|------------|--------------------------|
| `-port` | `:8081`    | Listen address           |
| `-db`   | `who3.db`  | SQLite database file path|

### API Endpoints

#### Health Check

```
GET /hello
```

```sh
curl http://localhost:8081/hello
```

Response:
```
Hello, World!
```

#### Report Game End

```
POST /game-end
Content-Type: application/json
```

Submit the list of players who just finished a game on a server. This increments their consecutive count and appends the map name. Players on the server who are **not** in the list get their streak reset to 0.

```sh
curl -X POST http://localhost:8081/game-end \
  -H "Content-Type: application/json" \
  -d '{
    "server_id": "1",
    "map_name": "campgrounds",
    "players": [
      { "steam_id": "12345567343", "name": "kgb" },
      { "steam_id": "22323232323", "name": "magdoll" }
    ]
  }'
```

Response:
```json
{"status":"ok"}
```

After a second game where only Alice stays:

```sh
curl -X POST http://localhost:8081/game-end \
  -H "Content-Type: application/json" \
  -d '{
    "server_id": "1",
    "map_name": "spider",
    "players": [
      { "steam_id": "12345567343", "name": "kgb" }
    ]
  }'
```

kgb now has `consecutive_count: 2` and `map_names: ["campgrounds", "spider"]`. magdoll is reset to 0.

#### Query Players

```
GET /players?server_id={id}&min={count}
```

| Parameter   | Required | Default | Description                                      |
|-------------|----------|---------|--------------------------------------------------|
| `server_id` | yes      | —       | Filter results to this server                    |
| `min`       | no       | `3`     | Minimum consecutive game count to include        |

```sh
# Players on a specific server with 3+ consecutive games (default min)
curl http://localhost:8081/players?server_id=1

# Players on a specific server with 2+ consecutive games
curl http://localhost:8081/players?server_id=1&min=2

# All servers, min=3 (server_id omitted)
curl http://localhost:8081/players
```

Response:
```json
[
  {
    "steam_id": "STEAM_0:1:12345",
    "name": "Alice",
    "consecutive_count": 5,
    "server_id": "1",
    "map_names": ["campgrounds", "bloodrun", "aerowalk", "dm6", "toxicity"]
  }
]
```

### Game-End Logic

All steps run in a single SQLite transaction:

1. **1-hour reset** — if the last game on this server was over 1 hour ago, reset all players on the server
2. **Absent-player reset** — players on the server not in the current game get their streak reset
3. **Increment active players** — upsert each player: count + 1, append map name, update name and timestamp
4. **Update server timestamp** — record when this game ended
