package leetify

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"

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
	playerData, pErr := c.fetchPlayerData(ctx, steamId)
	rawMatches, mErr := c.fetchPlayerMatches(ctx, steamId)

	var name *string

	var premierRating *int
	var wingmanRank *int
	var competitiveRanks []CompetitiveRank

	var leetifyRating *float64

	var matches *int
	var firstMatch *string

	var winRate *int

	var aimRating, positioning, utility *int
	var clutching, opening *float64

	var preAim *float64
	var reactionTime *int

	var kdRatio *float64

	var fallbackData map[string]any

	if pErr != nil || mErr != nil {
		// if fetching playerData or matches fails (no idea why it happens for some profiles),
		// we use leetify prod api (please dont sue me)

		// combine errors, probably a bad solution, subject to change
		combinedErr := errors.Join(pErr, mErr)

		var apiErr *fetcher.APIError
		if !errors.As(combinedErr, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return nil, fmt.Errorf("Failed fetching leetify data: %w", combinedErr)
		}

		var fbErr error
		fallbackData, fbErr = c.fetchFallbackData(ctx, steamId)
		if fbErr != nil {
			return nil, fmt.Errorf("Failed fetching fallback for leetify data: %w", fbErr)
		}
	}

	if pErr != nil {
		recentRatings, ok := fallbackData["recentGameRatings"].(map[string]any)
		if ok {
			if raw, ok := recentRatings["leetify"].(float64); ok {
				v := raw * 100
				leetifyRating = &v
			}

			if raw, ok := recentRatings["aim"].(float64); ok {
				v := int(math.Round(raw))
				aimRating = &v
			}

			if raw, ok := recentRatings["positioning"].(float64); ok {
				v := int(math.Round(raw))
				positioning = &v
			}

			if raw, ok := recentRatings["utility"].(float64); ok {
				v := int(math.Round(raw))
				utility = &v
			}

			if raw, ok := recentRatings["clutch"].(float64); ok {
				v := math.Round(raw*100*100) / 100
				clutching = &v
			}

			if raw, ok := recentRatings["opening"].(float64); ok {
				v := math.Round(raw*100*100) / 100
				opening = &v
			}
		}

		meta, ok := fallbackData["meta"].(map[string]any)
		if ok {
			name = utils.GetString(meta, "name")
		}
	} else {
		name = utils.GetString(playerData, "name")

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

		matches = utils.GetInt(playerData, "total_matches")
		firstMatch = utils.GetString(playerData, "first_match_date")

		if raw, ok := playerData["winrate"].(float64); ok {
			v := int(math.Round(raw * 100))
			winRate = &v
		}

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

		statsData, ok := playerData["stats"].(map[string]any)
		if ok {
			if raw, ok := statsData["preaim"].(float64); ok {
				v := math.Round(raw*100) / 100
				preAim = &v
			}
			reactionTime = utils.GetInt(statsData, "reaction_time_ms")
		}
	}

	if fallbackData != nil {
		games, ok := fallbackData["games"].([]any)
		if ok {
			if pErr != nil {
				gamesLen := len(games)
				matches = &gamesLen
				if gamesLen > 0 {
					firstMatchData, ok := games[gamesLen-1].(map[string]any)
					if ok {
						firstMatch = utils.GetString(firstMatchData, "gameFinishedAt")
					}
				}
			}

			totalWins := 0
			totalPreAim := 0.0
			totalReactionTime := 0
			totalKdRatio := 0.0
			matchCount := 0

			// prod api returns ALL the matches, we limit to 100 because thats the amount public api returns
			limit := min(len(games), 100)
			for _, g := range games[:limit] {
				game, ok := g.(map[string]any)
				if !ok {
					continue
				}

				kills := utils.GetFloat(game, "kills")
				deaths := utils.GetFloat(game, "deaths")
				if kills == nil || deaths == nil || *deaths == 0 {
					continue
				}

				if pErr != nil {
					rawScores, ok := game["scores"].([]any)
					if ok && len(rawScores) >= 2 {
						allyScore, ok0 := rawScores[0].(float64)
						enemyScore, ok1 := rawScores[1].(float64)
						if ok0 && ok1 && allyScore > enemyScore {
							totalWins++
						}
					}

					if raw, ok := game["preaim"].(float64); ok {
						v := math.Round(raw*100*100) / 100
						totalPreAim += v
					}

					gameReactionTime := utils.GetInt(game, "reactionTime")
					totalReactionTime += *gameReactionTime

				}

				if mErr != nil {
					totalKdRatio += *kills / *deaths
				}

				matchCount++
			}

			if matchCount > 0 {
				if pErr != nil {
					rawWinRate := int(math.Round(float64(totalWins) / float64(matchCount) * 100))
					winRate = &rawWinRate
					// for some users preaim and reaction time are 0 for some reason, just dont set these in that case
					if totalPreAim != 0 && totalReactionTime != 0 {
						rawPreAim := math.Round(totalPreAim/float64(matchCount)*100) / 100
						preAim = &rawPreAim
						rawReactionTime := totalReactionTime / matchCount
						reactionTime = &rawReactionTime
					}
				}
				if mErr != nil {
					kd := math.Round(totalKdRatio/float64(matchCount)*100) / 100
					kdRatio = &kd
				}
			}
		}
	}

	if mErr == nil {
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

func (c *Client) fetchFallbackData(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.cs-prod.leetify.com/api/profile/id/%s", steamID)
	return c.Fetch(ctx, url)
}
