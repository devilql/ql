package who3

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	DB *sql.DB
}

func (s *Server) HelloHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[hello] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodGet {
		log.Printf("[hello] rejected: method %s not allowed", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("Hello, World!"))
	log.Printf("[hello] 200 OK")
}

func (s *Server) GameEndHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[game-end] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		log.Printf("[game-end] rejected: method %s not allowed", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GameEndRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[game-end] bad request: invalid JSON: %v", err)
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[game-end] server=%s map=%s players=%d", req.ServerID, req.MapName, len(req.Players))
	for i, p := range req.Players {
		log.Printf("[game-end]   player[%d]: steam_id=%s name=%s", i, p.SteamID, p.Name)
	}

	if req.ServerID == "" {
		log.Printf("[game-end] bad request: missing server_id")
		http.Error(w, "server_id is required", http.StatusBadRequest)
		return
	}
	if req.MapName == "" {
		log.Printf("[game-end] bad request: missing map_name")
		http.Error(w, "map_name is required", http.StatusBadRequest)
		return
	}

	if err := ProcessGameEnd(s.DB, req, time.Now()); err != nil {
		log.Printf("[game-end] error processing game end: %v", err)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[game-end] 200 OK — processed successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) PlayersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[players] %s %s from %s", r.Method, r.URL.RequestURI(), r.RemoteAddr)
	if r.Method != http.MethodGet {
		log.Printf("[players] rejected: method %s not allowed", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	minStr := r.URL.Query().Get("min")
	minCount := 3 // default
	if minStr != "" {
		v, err := strconv.Atoi(minStr)
		if err != nil || v < 0 {
			log.Printf("[players] bad request: invalid min=%q", minStr)
			http.Error(w, "min must be a non-negative integer", http.StatusBadRequest)
			return
		}
		minCount = v
	}

	serverID := r.URL.Query().Get("server_id")

	log.Printf("[players] querying players with min=%d server_id=%q", minCount, serverID)
	players, err := GetPlayersByMinCount(s.DB, minCount, serverID)
	if err != nil {
		log.Printf("[players] error querying players: %v", err)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[players] 200 OK — returning %d player(s)", len(players))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}
