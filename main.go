package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

type Server struct {
	faceit  *FaceitClient
	leetify *LeetifyClient
	steam   *SteamClient
	cache   *Cache
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Failed to load .env file: %v", err)
	}

	c := &Server{
		faceit:  NewFaceitClient(os.Getenv("FACEIT_API_KEY")),
		leetify: NewLeetifyClient(os.Getenv("LEETIFY_API_KEY")),
		steam:   NewSteamClient(os.Getenv("STEAM_API_KEY")),
		cache:   NewCache(),
	}

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://steamcommunity.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/stats/faceit/{steamID}", c.handleFaceit)
	r.Get("/api/stats/leetify/{steamID}", c.handleLeetify)
	r.Get("/api/stats/steam/{steamID}", c.handleSteam)
	r.Get("/api/resolveVanity/{vanity}", c.handleSteamId)
	fmt.Println("Running on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Server error: %v", err)
		os.Exit(1)
	}
}

func (s *Server) handleSteam(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")
	w.Header().Set("Content-Type", "application/json")

	cacheKey := "steam:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.steam.getSteamProfile(r.Context(), steamID)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, profile, 60*time.Minute)

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleSteamId(w http.ResponseWriter, r *http.Request) {
	vanity := chi.URLParam(r, "vanity")
	w.Header().Set("Content-Type", "application/json")

	cacheKey := "steamid:" + vanity
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	steamID, err := s.steam.resolveVanity(r.Context(), vanity)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, map[string]string{"steam_id": steamID}, 60*time.Minute)

	writeJSON(w, http.StatusOK, map[string]string{"steam_id": steamID})
}

func (s *Server) handleLeetify(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")
	w.Header().Set("Content-Type", "application/json")

	cacheKey := "leetify:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.leetify.getProfile(r.Context(), steamID)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, profile, 5*time.Minute)

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleFaceit(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")
	w.Header().Set("Content-Type", "application/json")

	cacheKey := "faceit:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.faceit.getProfile(r.Context(), steamID)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, profile, 5*time.Minute)

	writeJSON(w, http.StatusOK, profile)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeApiError(w http.ResponseWriter, err error) {
  if apiErr, ok := errors.AsType[*APIError](err); ok {
    switch apiErr.StatusCode {
    case http.StatusNotFound:
      writeError(w, http.StatusNotFound, err)
    case http.StatusTooManyRequests:
      writeError(w, http.StatusTooManyRequests, err)
    default:
      writeError(w, http.StatusBadGateway, err)
    }
    return
  }
  writeError(w, http.StatusInternalServerError, err)
}
