package main

import (
	"context"
	"fmt"
	"time"
)

type SteamProfile struct {
	Name        string `json:"name"`
	Registered  string `json:"registered"`
	CS2Playtime int    `json:"cs2_playtime"`
}

type SteamClient struct {
	Fetcher
}

func NewSteamClient(apiKey string) *SteamClient {
	return &SteamClient{Fetcher: newFetcher(apiKey, "")}
}

func (c *SteamClient) getSteamProfile(ctx context.Context, steamID string) (*SteamProfile, error) {
	userData, err := c.fetchSteamUser(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching user data: %w", err)
	}

	response, _ := userData["response"].(map[string]any)
	players, _ := response["players"].([]any)
	if len(players) == 0 {
		return nil, fmt.Errorf("Failed to find steam profile with id: %s", steamID)
	}
	player, _ := players[0].(map[string]any)
	name, _ := player["personaname"].(string)
	rawRegistered, _ := player["timecreated"].(float64)
	registered := time.Unix(int64(rawRegistered), 0).UTC().Format("2006-01-02T15:04:05.000Z")

	gamesData, err := c.fetchSteamGames(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching games data: %w", err)
	}

	gamesResponse, _ := gamesData["response"].(map[string]any)
	gamesGames, _ := gamesResponse["games"].([]any)
	var rawPlaytime float64
	for _, g := range gamesGames {
		game, _ := g.(map[string]any)
		appid, _ := game["appid"].(float64)
		if int(appid) == 730 {
			rawPlaytime, _ = game["playtime_forever"].(float64)
			break
		}
	}
	// minutes into hours
	playtime := int(rawPlaytime) / 60

	return &SteamProfile{
		Name:        name,
		Registered:  registered,
		CS2Playtime: playtime,
	}, nil
}

func (c *SteamClient) fetchSteamUser(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", c.apiKey, steamID)
	return c.fetch(ctx, url)
}

func (c *SteamClient) fetchSteamGames(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s&appids_filter=730&language=en", c.apiKey, steamID)
	return c.fetch(ctx, url)
}
