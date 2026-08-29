package leetify

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
	"github.com/dom1torii/cs2-profilestats-api/internal/utils"
)

type Profile struct {
	Fallback bool    `json:"fallback"`
	Name     *string `json:"name"`
	Stats    *Stats  `json:"stats"`
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

var wingmanRankRegex = regexp.MustCompile(`\d+`)

func (c *Client) GetProfile(ctx context.Context, steamId string) (*Profile, error) {
	playerData, pErr := c.fetchPlayerData(ctx, steamId)
	rawMatches, mErr := c.fetchPlayerMatches(ctx, steamId)

	var fallback bool
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

	if pErr != nil {
		// if fetching playerData or matches fails (no idea why it happens for some profiles),
		// we use leetify extension api as a fallback (please dont sue me)
		fallback = true

		var apiErr *fetcher.APIError
		if !errors.As(pErr, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return nil, fmt.Errorf("failed fetching leetify profile: %w", pErr)
		}

		var fbErr error
		fallbackData, fbErr = c.fetchFallbackData(ctx, steamId)
		if fbErr != nil {
			return nil, fmt.Errorf("Failed fetching fallback for leetify data: %w", fbErr)
		}

		/* Missing: name, positioning, clutch, opening, first match, total matches */
		ratings, ok := fallbackData["ratings"].([]any)
		if ok {
			for _, r := range ratings {
				rating, ok := r.(map[string]any)
				if !ok {
					continue
				}

				title, ok := rating["title"].(string)
				if !ok {
					continue
				}

				switch title {
				case "Leetify Rating":
					leetifyRating = utils.GetFloatFromString(rating, "value")
				case "Aim Rating":
					aimRating = utils.GetIntFromString(rating, "value")
				case "Utility Rating":
					utility = utils.GetIntFromString(rating, "value")
				}
			}
		}

		winRateMap, ok := fallbackData["winRate"].(map[string]any)
		if ok {
			winRateValue := utils.GetString(winRateMap, "value")
			if winRateValue != nil {
				v, err := strconv.Atoi(strings.TrimSuffix(*winRateValue, "%"))
				if err != nil {
					winRate = nil
				} else {
					winRate = &v
				}
			}
		}

		aimRatingSkills, ok := fallbackData["aimRatingSkills"].([]any)
		if ok {
			for _, s := range aimRatingSkills {
				skill, ok := s.(map[string]any)
				if !ok {
					continue
				}

				title, ok := skill["title"].(string)
				if !ok {
					continue
				}

				switch title {
				case "Crosshair Placement":
					preAimValue := utils.GetString(skill, "value")
					if preAimValue != nil {
						v, err := strconv.ParseFloat(strings.TrimSuffix(*preAimValue, "°"), 64)
						if err != nil {
							preAim = nil
						} else {
							preAim = &v
						}

					}
				case "Time to Damage":
					reactionValue := utils.GetString(skill, "value")
					if reactionValue != nil {
						v, err := strconv.Atoi(strings.TrimSuffix(*reactionValue, "ms"))
						if err != nil {
							reactionTime = nil
						} else {
							reactionTime = &v
						}
					}
				}
			}
		}

		ranks, ok := fallbackData["ranks"].([]any)
		if ok {
			for _, r := range ranks {
				rank, ok := r.(map[string]any)
				if !ok {
					continue
				}

				source, ok := rank["dataSource"].(string)
				if !ok {
					continue
				}

				switch source {
				case "Premier":
					latest, ok := rank["latest"].(map[string]any)
					if ok {
						premierRating = utils.GetInt(latest, "value")
					}
				case "Wingman":
					latest, ok := rank["latest"].(map[string]any)
					if !ok {
						continue
					}
					url := utils.GetString(latest, "imageUrl")
					if url == nil {
						continue
					}
					match := wingmanRankRegex.FindString(*url)
					v, err := strconv.Atoi(match)
					if err != nil {
						wingmanRank = nil
					} else {
						wingmanRank = &v
					}
				}
			}
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

	if mErr != nil {
		var apiErr *fetcher.APIError
		if !errors.As(mErr, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return nil, fmt.Errorf("failed fetching leetify matches: %w", mErr)
		}
		// only fetch fallbackData here if we don't already have it from the pErr branch
		if fallbackData == nil {
			var fbErr error
			fallbackData, fbErr = c.fetchFallbackData(ctx, steamId)
			if fbErr != nil {
				return nil, fmt.Errorf("failed fetching fallback for leetify data: %w", fbErr)
			}
		}
		kdMap, ok := fallbackData["kdRatio"].(map[string]any)
		if ok {
			kdRatio = utils.GetFloatFromString(kdMap, "value")
		}
	} else {
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
		Fallback: fallback,
		Name:     name,
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
	url := fmt.Sprintf("https://api.cs-prod.leetify.com/api/profile/%s/browser-extension-summary", steamID)
	return c.Fetch(ctx, url)
}
