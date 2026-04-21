package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

 	fileServer := http.FileServer(http.Dir("./assets"))
  r.Handle("/assets/*", http.StripPrefix("/assets", fileServer))

	r.Get("/api/status", s.handleStatus)
	r.Get("/api/stats/faceit/{steamID}", s.handleFaceit)
	r.Get("/api/stats/leetify/{steamID}", s.handleLeetify)
	r.Get("/api/stats/steam/{steamID}", s.handleSteam)
	r.Get("/api/resolveVanity/{vanity}", s.handleSteamId)

	return r
}
