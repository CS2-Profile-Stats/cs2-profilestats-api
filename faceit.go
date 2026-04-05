package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type FaceitProfile struct {
	PlayerID   string      `json:"player_id"`
	Nickname   string      `json:"nickname"`
	Avatar     string      `json:"avatar"`
	Country    string      `json:"country"`
	Registered string      `json:"registered"`
	ProfileUrl string      `json:"profile_url"`
	Level      int         `json:"level"`
	Elo        int         `json:"elo"`
	Ranking    int         `json:"ranking"`
	Membership string      `json:"membership"`
	Stats      FaceitStats `json:"stats"`
}

type FaceitStats struct {
	Matches       int      `json:"matches"`
	KDRatio       float64  `json:"kd_ratio"`
	HSPercentage  int      `json:"hs_percentage"`
	WinRate       int      `json:"win_rate"`
	RecentResults []string `json:"recent_results"`
	AvgKills      int      `json:"avg_kills"`
}

type FaceitClient struct {
	Fetcher
}

func NewFaceitClient(apiKey string) *FaceitClient {
	return &FaceitClient{Fetcher: newFetcher("Bearer "+apiKey, "Authorization")}
}

func (c *FaceitClient) getProfile(ctx context.Context, steamId string) (*FaceitProfile, error) {
	playerData, err := c.fetchPlayerData(ctx, steamId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player data: %w", err)
	}

	playerId, _ := playerData["player_id"].(string)
	nickname, _ := playerData["nickname"].(string)
	avatar, _ := playerData["avatar"].(string)
	country, _ := playerData["country"].(string)
	registered, _ := playerData["activated_at"].(string)
	rawProfileUrl, _ := playerData["faceit_url"].(string)
	profileUrl := strings.Replace(rawProfileUrl, "{lang}", "en", 1)
	games, _ := playerData["games"].(map[string]any)
	cs2, _ := games["cs2"].(map[string]any)
	region, _ := cs2["region"].(string)
	rawLevel, _ := cs2["skill_level"].(float64)
	level := int(rawLevel)
	rawElo, _ := cs2["faceit_elo"].(float64)
	elo := int(rawElo)
	memberships, _ := playerData["memberships"].([]any)
	membership, _ := memberships[0].(string)

	playerStats, err := c.fetchPlayerStats(ctx, playerId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player stats: %w", err)
	}

	lifetime, _ := playerStats["lifetime"].(map[string]any)
	matches, _ := strconv.Atoi(lifetime["Matches"].(string))
	kdRatio, _ := strconv.ParseFloat(lifetime["Average K/D Ratio"].(string), 64)
	hsPercentage, _ := strconv.Atoi(lifetime["Average Headshots %"].(string))
	winRate, _ := strconv.Atoi(lifetime["Win Rate %"].(string))
	rawRecentResults, _ := lifetime["Recent Results"].([]any)
	recentResults := make([]string, len(rawRecentResults))
	for i, v := range lifetime["Recent Results"].([]any) {
		if v.(string) == "1" {
			recentResults[i] = "W"
		} else {
			recentResults[i] = "L"
		}
	}
	segments, _ := playerStats["segments"].([]any)
	totalAvgKills := 0.0
	for _, s := range segments {
		segment := s.(map[string]any)
		stats := segment["stats"].(map[string]any)
		avgKills, _ := strconv.ParseFloat(stats["Average Kills"].(string), 64)
		totalAvgKills += avgKills
	}
	avgKills := int(totalAvgKills / float64(len(segments)))

	playerRanking, err := c.fetchPlayerRanking(ctx, region, playerId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player ranking: %w", err)
	}

	rawRanking := playerRanking["position"].(float64)
	ranking := int(rawRanking)


	return &FaceitProfile{
		PlayerID:   playerId,
		Nickname:   nickname,
		Avatar:     avatar,
		Country:    country,
		Registered: registered,
		ProfileUrl: profileUrl,
		Level:      level,
		Elo:        elo,
		Ranking:    ranking,
		Membership: membership,
		Stats: FaceitStats{
			Matches:       matches,
			KDRatio:       kdRatio,
			HSPercentage:  hsPercentage,
			WinRate:       winRate,
			RecentResults: recentResults,
			AvgKills:      avgKills,
		},
	}, nil
}

func (c *FaceitClient) fetchPlayerData(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players?game=cs2&game_player_id=%s", steamID)
	return c.fetch(ctx, url)
}

func (c *FaceitClient) fetchPlayerStats(ctx context.Context, playerID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/stats/cs2", playerID)
	return c.fetch(ctx, url)
}

func (c *FaceitClient) fetchPlayerRanking(ctx context.Context, region string, playerID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/rankings/games/cs2/regions/%s/players/%s", region, playerID)
	return c.fetch(ctx, url)
}
