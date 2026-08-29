package csrep

import (
	"context"
	"fmt"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
	"github.com/dom1torii/cs2-profilestats-api/internal/utils"
)

type Profile struct {
	Name     *string `json:"name"`
	Redacted *bool   `json:"redacted"`
	Stats    *Stats  `json:"stats"`
}

type Stats struct {
	PremierRating    *int              `json:"premier_rating"`
	TrustRating      *int              `json:"trust_rating"`
	CompetitiveRanks []CompetitiveRank `json:"competitive_ranks"`
}

type CompetitiveRank struct {
	Map  *string `json:"map"`
	Rank *int    `json:"rank"`
}

type Client struct {
	fetcher.Fetcher
}

func NewClient(apiKey string) *Client {
	return &Client{Fetcher: fetcher.New(apiKey, "X-API-Key")}
}

func (c *Client) GetProfile(ctx context.Context, steamId string) (*Profile, error) {
	playerData, err := c.fetchPlayerData(ctx, steamId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player data: %w", err)
	}

	var name *string
	var redacted *bool
	var premierRating *int
	var trustRating *int
	var competitiveRanks []CompetitiveRank
	result, ok := playerData["result"].(map[string]any)
	if ok {
		name = utils.GetString(result, "name")
		redacted = utils.GetBool(result, "redacted")
		premierRating = utils.GetInt(result, "premier_elo")
		trustRating = utils.GetInt(result, "trust_rating")

		rawCompetitive, rcOk := result["map_ranks"].(map[string]any)
		if rcOk {
			for mapName := range rawCompetitive {
				rankInt := utils.GetInt(rawCompetitive, mapName)
				competitiveRanks = append(competitiveRanks, CompetitiveRank{
					Map:  &mapName,
					Rank: rankInt,
				})
			}
		}
	}

	return &Profile{
		Name:     name,
		Redacted: redacted,
		Stats: &Stats{
			PremierRating:    premierRating,
			TrustRating:      trustRating,
			CompetitiveRanks: competitiveRanks,
		},
	}, nil
}

func (c *Client) fetchPlayerData(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://csrep.gg/api/players/%s", steamID)
	return c.Fetch(ctx, url)
}
