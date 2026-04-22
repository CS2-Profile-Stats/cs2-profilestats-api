package leetify

import (
	"context"
	"fmt"
	"math"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
	"github.com/dom1torii/cs2-profilestats-api/internal/utils"
)

type Profile struct {
	Name  *string `json:"name"`
	Stats *Stats  `json:"stats"`
}

type Stats struct {
	LeetifyRating    *float64          `json:"leetify_rating"`
	PremierRating    *int              `json:"premier_rating"`
	CompetitiveRanks []CompetitiveRank `json:"competitive_ranks"`
	WingmanRank      *int              `json:"wingman_rank"`
	Matches          *int              `json:"matches"`
	FirstMatch       *string           `json:"first_match"`
	KDRatio          *float64          `json:"kd_ratio"`
	WinRate          *int              `json:"win_rate"`
	AimRating        *int              `json:"aim_rating"`
	Positioning      *int              `json:"positioning"`
	Utility          *int              `json:"utility"`
	Clutching        *float64          `json:"clutching"`
	Opening          *float64          `json:"opening"`
	PreAim           *float64          `json:"preaim_angle"`
	ReactionTime     *int              `json:"reaction_time"`
}

type CompetitiveRank struct {
	Map  *string `json:"map"`
	Rank *int    `json:"rank"`
}

type Client struct {
	fetcher.Fetcher
}

func NewClient(apiKey string) *Client {
	return &Client{Fetcher: fetcher.New(apiKey, "_leetify_key")}
}

func (c *Client) GetProfile(ctx context.Context, steamId string) (*Profile, error) {
	playerData, err := c.fetchPlayerData(ctx, steamId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player data: %w", err)
	}

	name := utils.GetString(playerData, "name")

	var leetifyRating *float64
	var premierRating *int
	var wingmanRank *int
	var competitiveRanks []CompetitiveRank
	ranks, ok := playerData["ranks"].(map[string]any)
	if ok {
		if raw, ok := ranks["leetify"].(float64); ok {
			v := math.Round(raw*100) / 100
			leetifyRating = &v
		}
		premierRating = utils.GetInt(ranks, "premier")
		wingmanRank = utils.GetInt(ranks, "wingman")

		rawCompetitive, ok := ranks["competitive"].([]any)
		if ok && len(rawCompetitive) > 0 {
			competitiveRanks = make([]CompetitiveRank, 0, len(rawCompetitive))
			for _, v := range rawCompetitive {
				entry, ok := v.(map[string]any)
				if !ok {
					continue
				}
				competitiveRanks = append(competitiveRanks, CompetitiveRank{
					Map:  utils.GetString(entry, "map_name"),
					Rank: utils.GetInt(entry, "rank"),
				})
			}
		}
	}

	matches := utils.GetInt(playerData, "total_matches")
	firstMatch := utils.GetString(playerData, "first_match_date")

	var winRate *int
	if raw, ok := playerData["winrate"].(float64); ok {
		v := int(raw * 100)
		winRate = &v
	}

	var aimRating, positioning, utility *int
	var clutching, opening *float64
	rating, ok := playerData["rating"].(map[string]any)
	if ok {
		aimRating = utils.GetInt(rating, "aim")
		positioning = utils.GetInt(rating, "positioning")
		utility = utils.GetInt(rating, "utility")

		if raw, ok := rating["clutch"].(float64); ok {
			v := math.Round(raw*100*100) / 100
			clutching = &v
		}
		if raw, ok := rating["opening"].(float64); ok {
			v := math.Round(raw*100*100) / 100
			opening = &v
		}
	}

	var preAim *float64
	var reactionTime *int
	statsData, ok := playerData["stats"].(map[string]any)
	if ok {
		if raw, ok := statsData["preaim"].(float64); ok {
			v := math.Round(raw*100) / 100
			preAim = &v
		}
		reactionTime = utils.GetInt(statsData, "reaction_time_ms")
	}

	rawMatches, err := c.fetchPlayerMatches(ctx, steamId)
	if err != nil {
    return nil, fmt.Errorf("Failed fetching matches: %w", err)
	}

	var kdRatio *float64
	totalKdRatio := 0.0
	matchCount := 0
	for _, m := range rawMatches {
    match, ok := m.(map[string]any)
    if !ok {
      continue
    }
    stats, ok := match["stats"].([]any)
    if !ok {
      continue
    }
    for _, s := range stats {
      stat, ok := s.(map[string]any)
      if !ok {
        continue
      }
      kd := utils.GetFloat(stat, "kd_ratio")
      if kd == nil {
        continue
      }
      totalKdRatio += *kd
      matchCount++
    }
	}
	if matchCount > 0 {
    kd := math.Round(totalKdRatio/float64(matchCount)*100) / 100
    kdRatio = &kd
	}

	return &Profile{
		Name: name,
		Stats: &Stats{
			LeetifyRating:    leetifyRating,
			PremierRating:    premierRating,
			CompetitiveRanks: competitiveRanks,
			WingmanRank:      wingmanRank,
			Matches:          matches,
			FirstMatch:       firstMatch,
			WinRate:          winRate,
			KDRatio:          kdRatio,
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

func (c *Client) fetchPlayerData(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api-public.cs-prod.leetify.com/v3/profile?steam64_id=%s", steamID)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchPlayerMatches(ctx context.Context, steamID string) ([]any, error) {
	url := fmt.Sprintf("https://api-public.cs-prod.leetify.com/v3/profile/matches?steam64_id=%s", steamID)
	return c.FetchArray(ctx, url)
}
