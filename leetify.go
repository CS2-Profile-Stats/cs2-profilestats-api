package main

import (
	"context"
	"fmt"
	"math"
)

type LeetifyProfile struct {
	Name  string       `json:"name"`
	Stats LeetifyStats `json:"stats"`
}

type LeetifyStats struct {
	PremierRating    int               `json:"premier_rating"`
	CompetitiveRanks []CompetitiveRank `json:"competitive_ranks"`
	Matches          int               `json:"matches"`
	FirstMatch       string            `json:"first_match"`
	WinRate          int               `json:"win_rate"`
	AimRating        int               `json:"aim_rating"`
	Positioning      int               `json:"positioning"`
	Utility          int               `json:"utility"`
	Clutching        float64           `json:"clutching"`
	Opening          float64           `json:"opening"`
	PreAim           float64           `json:"preaim_angle"`
	ReactionTime     int               `json:"reaction_time"`
}

type CompetitiveRank struct {
	Map  string `json:"map"`
	Rank int    `json:"rank"`
}

type LeetifyClient struct {
	Fetcher
}

func NewLeetifyClient(apiKey string) *LeetifyClient {
	return &LeetifyClient{Fetcher: newFetcher(apiKey, "_leetify_key")}
}

func (c *LeetifyClient) getProfile(ctx context.Context, steamId string) (*LeetifyProfile, error) {
	playerData, err := c.fetchPlayerData(ctx, steamId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player data: %w", err)
	}

	name, _ := playerData["name"].(string)
	ranks, _ := playerData["ranks"].(map[string]any)
	rawPremierRating, _ := ranks["premier"].(float64)
	premierRating := int(rawPremierRating)
	rawCompetitive := ranks["competitive"].([]any)
	competitiveRanks := make([]CompetitiveRank, 0, len(rawCompetitive))
	for _, v := range rawCompetitive {
		entry, _ := v.(map[string]any)
		mapName, _ := entry["map_name"].(string)
		rawRank, _ := entry["rank"].(float64)
		competitiveRanks = append(competitiveRanks, CompetitiveRank{
			Map:  mapName,
			Rank: int(rawRank),
		})
	}
	rawMatches, _ := playerData["total_matches"].(float64)
	matches := int(rawMatches)
	firstMatch, _ := playerData["first_match_date"].(string)
	rawWinRate, _ := playerData["winrate"].(float64)
	winRate := int(rawWinRate * 100)
	rating, _ := playerData["rating"].(map[string]any)
	rawAimRating, _ := rating["aim"].(float64)
	aimRating := int(rawAimRating)
	rawPositioning, _ := rating["positioning"].(float64)
	positioning := int(rawPositioning)
	rawUtility, _ := rating["utility"].(float64)
	utility := int(rawUtility)
	rawClutching, _ := rating["clutch"].(float64)
	clutching := math.Round(rawClutching*100*100) / 100
	rawOpening, _ := rating["opening"].(float64)
	opening := math.Round(rawOpening*100*100) / 100
	stats, _ := playerData["stats"].(map[string]any)
	rawPreAim, _ := stats["preaim"].(float64)
	preAim := math.Round(rawPreAim*100) / 100
	rawReactionTime, _ := stats["reaction_time_ms"].(float64)
	reactionTime := int(rawReactionTime)

	return &LeetifyProfile{
		Name: name,
		Stats: LeetifyStats{
			PremierRating:    premierRating,
			CompetitiveRanks: competitiveRanks,
			Matches:          matches,
			FirstMatch:       firstMatch,
			WinRate:          winRate,
			AimRating:        aimRating,
			Positioning:      positioning,
			Utility:          utility,
			Clutching:        clutching,
			Opening:          opening,
			PreAim:           preAim,
			ReactionTime:     reactionTime,
		},
	}, nil
}

func (c *LeetifyClient) fetchPlayerData(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api-public.cs-prod.leetify.com/v3/profile?steam64_id=%s", steamID)
	return c.fetch(ctx, url)
}
