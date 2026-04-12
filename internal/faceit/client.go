package faceit

import (
	"context"
	"fmt"
	"strings"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
	"github.com/dom1torii/cs2-profilestats-api/internal/utils"
)

type Profile struct {
	PlayerID   *string `json:"player_id"`
	Nickname   *string `json:"nickname"`
	Banned     *bool   `json:"banned"`
	BanReason  *string `json:"ban_reason"`
	Avatar     *string `json:"avatar"`
	Country    *string `json:"country"`
	Registered *string `json:"registered"`
	ProfileUrl *string `json:"profile_url"`
	Level      *int    `json:"level"`
	Elo        *int    `json:"elo"`
	Ranking    *int    `json:"ranking"`
	Membership *string `json:"membership"`
	Stats      *Stats  `json:"stats"`
}

type Stats struct {
	Matches       *int      `json:"matches"`
	KDRatio       *float64  `json:"kd_ratio"`
	HSPercentage  *int      `json:"hs_percentage"`
	WinRate       *int      `json:"win_rate"`
	RecentResults []*string `json:"recent_results"`
	AvgKills      *int      `json:"avg_kills"`
}

type Client struct {
	fetcher.Fetcher
}

func NewClient(apiKey string) *Client {
	return &Client{Fetcher: fetcher.New("Bearer "+apiKey, "Authorization")}
}

func (c *Client) GetProfile(ctx context.Context, game string, steamId string) (*Profile, error) {
	playerData, err := c.fetchPlayerData(ctx, game, steamId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player data: %w", err)
	}

	playerId := utils.GetString(playerData, "player_id")
	if playerId == nil {
		return nil, fmt.Errorf("player_id field is missing")
	}

	nickname := utils.GetString(playerData, "nickname")
	avatar := utils.GetString(playerData, "avatar")
	country := utils.GetString(playerData, "country")
	registered := utils.GetString(playerData, "activated_at")
	rawProfileUrl := utils.GetString(playerData, "faceit_url")

	var profileUrl *string
	if rawProfileUrl != nil {
		replaced := strings.Replace(*rawProfileUrl, "{lang}", "en", 1)
		profileUrl = &replaced
	}

	var region *string
	var level, elo *int
	games, ok := playerData["games"].(map[string]any)
	if ok {
		cs2, ok := games[game].(map[string]any)
		if ok {
			region = utils.GetString(cs2, "region")
			level = utils.GetInt(cs2, "skill_level")
			elo = utils.GetInt(cs2, "faceit_elo")
		}
	}

	var membership *string
	memberships, ok := playerData["memberships"].([]any)
	if ok && len(memberships) > 0 {
		v, _ := memberships[0].(string)
		if v == "super_match_token" {
			v = "free"
		}
		membership = &v
	}

	playerBans, err := c.fetchPlayerBans(ctx, *playerId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player bans: %w", err)
	}

	var banned *bool
	var banReason *string
	items, _ := playerBans["items"].([]any)
	if len(items) > 0 {
		b := true
		banned = &b
		ban, _ := items[0].(map[string]any)
		banReason = utils.GetString(ban, "reason")
	} else {
		b := false
		banned = &b
	}

	playerStats, err := c.fetchPlayerStats(ctx, game, *playerId)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching player stats: %w", err)
	}

	lifetime, _ := playerStats["lifetime"].(map[string]any)
	matches := utils.GetIntFromString(lifetime, "Matches")
	kdRatio := utils.GetFloatFromString(lifetime, "Average K/D Ratio")
	hsPercentage := utils.GetIntFromString(lifetime, "Average Headshots %")
	winRate := utils.GetIntFromString(lifetime, "Win Rate %")

	var recentResults []*string
	rawRecentResults, _ := lifetime["Recent Results"].([]any)
	if len(rawRecentResults) > 0 {
		recentResults = make([]*string, len(rawRecentResults))
		for i, v := range rawRecentResults {
			val, _ := v.(string)
			var result string
			if val == "1" {
				result = "W"
			} else {
				result = "L"
			}
			recentResults[i] = &result
		}
	}

	segments, _ := playerStats["segments"].([]any)
	totalAvgKills := 0.0
	avgCount := 0
	var avgKills *int
	for _, s := range segments {
		segment, ok := s.(map[string]any)
		if !ok {
			continue
		}
		stats, ok := segment["stats"].(map[string]any)
		if !ok {
			continue
		}
		avg := utils.GetFloatFromString(stats, "Average Kills")
		if avg == nil {
			continue
		}
		totalAvgKills += *avg
		avgCount++
	}
	if avgCount > 0 {
		v := int(totalAvgKills / float64(avgCount))
		avgKills = &v
	}

	var ranking *int

	if region != nil {
		playerRanking, err := c.fetchPlayerRanking(ctx, game, *region, *playerId)
		if err != nil {
			return nil, fmt.Errorf("Failed fetching player ranking: %w", err)
		}
		ranking = utils.GetInt(playerRanking, "position")
	}

	return &Profile{
		PlayerID:   playerId,
		Nickname:   nickname,
		Banned:     banned,
		BanReason:  banReason,
		Avatar:     avatar,
		Country:    country,
		Registered: registered,
		ProfileUrl: profileUrl,
		Level:      level,
		Elo:        elo,
		Ranking:    ranking,
		Membership: membership,
		Stats: &Stats{
			Matches:       matches,
			KDRatio:       kdRatio,
			HSPercentage:  hsPercentage,
			WinRate:       winRate,
			RecentResults: recentResults,
			AvgKills:      avgKills,
		},
	}, nil
}

func (c *Client) fetchPlayerData(ctx context.Context, game string, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players?game=%s&game_player_id=%s", game, steamID)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchPlayerStats(ctx context.Context, game string, playerID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/stats/%s", playerID, game)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchPlayerRanking(ctx context.Context, game string, region string, playerID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/rankings/games/%s/regions/%s/players/%s", game, region, playerID)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchPlayerBans(ctx context.Context, playerID string) (map[string]any, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/bans", playerID)
	return c.Fetch(ctx, url)
}
