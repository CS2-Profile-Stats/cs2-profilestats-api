package server

import (
	"fmt"
	"net/http"

	"github.com/dom1torii/cs2-profilestats-api/internal/cache"
	"github.com/dom1torii/cs2-profilestats-api/internal/csstats"
	"github.com/dom1torii/cs2-profilestats-api/internal/faceit"
	"github.com/dom1torii/cs2-profilestats-api/internal/leetify"
	"github.com/dom1torii/cs2-profilestats-api/internal/steam"
)

type Server struct {
	faceit  *faceit.Client
	leetify *leetify.Client
	steam   *steam.Client
	csstats *csstats.Client
	cache   *cache.Cache
}

func New(f *faceit.Client, l *leetify.Client, s *steam.Client, cs *csstats.Client, c *cache.Cache) *Server {
	return &Server{
		faceit:  f,
		leetify: l,
		steam:   s,
		csstats: cs,
		cache:   c,
	}
}

func (s *Server) Run(addr string) error {
	fmt.Printf("Running on %s\n", addr)
	return http.ListenAndServe(addr, s.routes())
}
