package steam

import (
	"context"
	"fmt"
	"time"

	"github.com/dom1torii/cs2-profilestats-api/internal/fetcher"
	"github.com/dom1torii/cs2-profilestats-api/internal/utils"
)

type Profile struct {
	Name              *string `json:"name"`
	Registered        *string `json:"registered"`
	CS2Playtime       *int    `json:"cs2_playtime"`
	CS2Playtime2Weeks *int    `json:"cs2_playtime_2weeks"`
	CommunityBanned   *bool   `json:"community_banned"`
}

type Client struct {
	fetcher.Fetcher
	apiKey string
}

func NewClient(apiKey string) *Client {
	return &Client{
		Fetcher: fetcher.New(apiKey, ""),
		apiKey:  apiKey,
	}
}

func (c *Client) GetProfile(ctx context.Context, steamID string) (*Profile, error) {
	userData, err := c.fetchSteamUser(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching user data: %w", err)
	}

	response, ok := userData["response"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response field is missing")
	}

	players, ok := response["players"].([]any)
	if !ok || len(players) == 0 {
		return nil, fmt.Errorf("Failed to find steam profile with id: %s", steamID)
	}

	player, ok := players[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Player data is missing")
	}

	name := utils.GetString(player, "personaname")

	var registered *string
	if raw, ok := player["timecreated"].(float64); ok {
		v := time.Unix(int64(raw), 0).UTC().Format("2006-01-02T15:04:05.000Z")
		registered = &v
	}

	gamesData, err := c.fetchSteamGames(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching games data: %w", err)
	}

	var game map[string]any
	if response, ok := gamesData["response"].(map[string]any); ok {
		if games, ok := response["games"].([]any); ok && len(games) > 0 {
			game, _ = games[0].(map[string]any)
		}
	}

	playtime := minToH(game, "playtime_forever")
	playtime2Weeks := minToH(game, "playtime_2weeks")

	bansData, err := c.fetchUserBans(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("Failed fetching user's bans: %w", err)
	}

	bansPlayers, ok := bansData["players"].([]any)
	if !ok || len(bansPlayers) == 0 {
		return nil, fmt.Errorf("Failed to find bans for user: %s", steamID)
	}

	bansPlayer, ok := bansPlayers[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Player bans data is missing")
	}

	var communityBanned *bool
	if raw, ok := bansPlayer["CommunityBanned"].(bool); ok {
		communityBanned = &raw
	}

	return &Profile{
		Name:              name,
		Registered:        registered,
		CS2Playtime:       playtime,
		CS2Playtime2Weeks: playtime2Weeks,
		CommunityBanned:   communityBanned,
	}, nil
}

func (c *Client) fetchSteamUser(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", c.apiKey, steamID)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchSteamGames(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s&appids_filter[0]=730&include_played_free_games=true&language=en", c.apiKey, steamID)
	return c.Fetch(ctx, url)
}

func (c *Client) fetchUserBans(ctx context.Context, steamID string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerBans/v1/?key=%s&steamids=%s", c.apiKey, steamID)
	return c.Fetch(ctx, url)
}

func (c *Client) ResolveVanity(ctx context.Context, vanity string) (string, error) {
	steamIDData, err := c.fetchSteamID64(ctx, vanity)
	if err != nil {
		return "", fmt.Errorf("Failed fetching steamid64: %w", err)
	}
	response, ok := steamIDData["response"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing response field")
	}
	steamID, ok := response["steamid"].(string)
	if !ok {
		return "", fmt.Errorf("Failed to resolve vanity: %s", vanity)
	}
	return steamID, nil
}

func (c *Client) fetchSteamID64(ctx context.Context, vanity string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s&url_type=1", c.apiKey, vanity)
	return c.Fetch(ctx, url)
}

func minToH(game map[string]any, key string) *int {
	if game == nil {
		return nil
	}
	raw, ok := game[key].(float64)
	if !ok {
		return nil
	}
	v := int(raw) / 60
	return &v
}
