package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"

	"github.com/dom1torii/cs2-profilestats-api/internal/cache"
	"github.com/dom1torii/cs2-profilestats-api/internal/faceit"
	"github.com/dom1torii/cs2-profilestats-api/internal/leetify"
	"github.com/dom1torii/cs2-profilestats-api/internal/server"
	"github.com/dom1torii/cs2-profilestats-api/internal/steam"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			fmt.Printf("Failed to load .env file: %v\n", err)
		}
	}

	s := server.New(
		faceit.NewClient(os.Getenv("FACEIT_API_KEY")),
		leetify.NewClient(os.Getenv("LEETIFY_API_KEY")),
		steam.NewClient(os.Getenv("STEAM_API_KEY")),
		cache.New(),
	)

	if err := s.Run(":8080"); err != nil {
		fmt.Printf("Server error: %v", err)
		os.Exit(1)
	}
}
