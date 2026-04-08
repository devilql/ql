package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/devilql/ql/internal/who3"
)

func main() {
	port := flag.String("port", ":8081", "listen address")
	dbPath := flag.String("db", "who3.db", "SQLite database file path")
	flag.Parse()

	db, err := who3.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	srv := &who3.Server{DB: db}

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", srv.HelloHandler)
	mux.HandleFunc("/game-end", srv.GameEndHandler)
	mux.HandleFunc("/players", srv.PlayersHandler)

	log.Printf("who3 listening on %s", *port)
	if err := http.ListenAndServe(*port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
