package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
  writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSteam(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")

	cacheKey := "steam:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.steam.GetProfile(r.Context(), steamID)
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

	steamID, err := s.steam.ResolveVanity(r.Context(), vanity)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, map[string]string{"steam_id": steamID}, 60*time.Minute)

	writeJSON(w, http.StatusOK, map[string]string{"steam_id": steamID})
}

func (s *Server) handleLeetify(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")

	cacheKey := "leetify:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.leetify.GetProfile(r.Context(), steamID)
	if err != nil {
		writeApiError(w, err)
		return
	}

	s.cache.Set(cacheKey, profile, 5*time.Minute)

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleFaceit(w http.ResponseWriter, r *http.Request) {
  steamID := chi.URLParam(r, "steamID")

  game := r.URL.Query().Get("game")
  if game == "" {
    game = "cs2"
  }

  cacheKey := "faceit:" + game + ":" + steamID
  if cached, ok := s.cache.Get(cacheKey); ok {
    writeJSON(w, http.StatusOK, cached)
    return
  }

  profile, err := s.faceit.GetProfile(r.Context(), game, steamID)
  if err != nil {
    writeApiError(w, err)
    return
  }

  s.cache.Set(cacheKey, profile, 5*time.Minute)

  writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleCsstats(w http.ResponseWriter, r *http.Request) {
	steamID := chi.URLParam(r, "steamID")

	cacheKey := "csstats:" + steamID
	if cached, ok := s.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	profile, err := s.csstats.GetProfile(r.Context(), steamID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.cache.Set(cacheKey, profile, 30*time.Minute)

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
	if apiErr, ok := errors.AsType[*fetcher.APIError](err); ok {
		switch apiErr.StatusCode {
		case http.StatusNotFound:
			writeError(w, http.StatusNotFound, apiErr)
		case http.StatusTooManyRequests:
			writeError(w, http.StatusTooManyRequests, apiErr)
		default:
			writeError(w, http.StatusBadGateway, apiErr)
		}
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}
