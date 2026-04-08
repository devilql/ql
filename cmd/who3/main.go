package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	CORSOrigins := parseCSV(getEnv("CORS_ORIGINS", ""))
	CORSOriginSuffixes := parseCSV(getEnv("CORS_ORIGIN_SUFFIXES", ""))

	allowedOrigins := buildAllowedOrigins(CORSOrigins, CORSOriginSuffixes)
	log.Printf("CORS exact origins configured: %d", len(allowedOrigins.exactOrigins))
	if len(allowedOrigins.hostSuffixes) > 0 {
		log.Printf("CORS host suffixes configured: %v", allowedOrigins.hostSuffixes)
	}
	muxWithCors := corsMiddleware(allowedOrigins)(mux)

	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)

	log.Printf("who3 listening on %s", *port)
	if err := http.ListenAndServe(*port, muxWithCors); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

type corsOriginMatcher struct {
	exactOrigins map[string]bool
	hostSuffixes []string
}

// buildAllowedOrigins constructs the CORS allowlist. localhost entries are always
// included so local development works without any environment configuration.
// Additional exact origins and host suffix rules can be configured via environment variables.
func buildAllowedOrigins(exactOrigins []string, suffixes []string) corsOriginMatcher {
	exactOriginMap := map[string]bool{
		"http://localhost":      true, // bare localhost (e.g. Tauri desktop app)
		"http://localhost:5174": true, // SvelteKit dev server
	}
	for _, origin := range exactOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			exactOriginMap[trimmed] = true
		}
	}

	normalizedSuffixes := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(suffix, "*.")))
		normalized = strings.TrimPrefix(normalized, ".")
		if normalized != "" {
			normalizedSuffixes = append(normalizedSuffixes, normalized)
		}
	}

	return corsOriginMatcher{
		exactOrigins: exactOriginMap,
		hostSuffixes: normalizedSuffixes,
	}
}

func (m corsOriginMatcher) isAllowed(origin string) bool {
	if origin == "" {
		return false
	}

	if m.exactOrigins[origin] {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return false
	}

	for _, suffix := range m.hostSuffixes {
		if hostname == suffix || strings.HasSuffix(hostname, "."+suffix) {
			return true
		}
	}

	return false
}

// corsMiddleware adds CORS headers to allow frontend access.
// It only sets Access-Control-Allow-Origin for origins in the allowlist,
// rejecting all others by omitting the header entirely (the browser then blocks the request).
func corsMiddleware(allowedOrigins corsOriginMatcher) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigins.isAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Authorization")
				w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version")
				// Vary: Origin tells HTTP caches that the response differs based on the
				// Origin request header. Without this, a cache could store a response
				// that includes Access-Control-Allow-Origin: https://devstats-2lk.pages.dev
				// and then serve that cached response to a different origin (e.g. localhost),
				// which would either incorrectly grant or deny access. By declaring Vary: Origin,
				// each distinct Origin gets its own cache entry, keeping CORS behaviour correct.
				w.Header().Set("Vary", "Origin")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseCSV parses comma-separated values, trimming whitespace and skipping empties.
func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
