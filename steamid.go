package main

import (
	"context"
	"fmt"
)

func (c *SteamClient) resolveVanity(ctx context.Context, vanity string) (string, error) {
	steamIDData, err := c.fetchSteamID64(ctx, vanity)
	if err != nil {
		return "", fmt.Errorf("Failed fetching steamid64: %w", err)
	}

	response, _ := steamIDData["response"].(map[string]any)
	steamID, _ := response["steamid"].(string)

	return steamID, nil
}

func (c *SteamClient) fetchSteamID64(ctx context.Context, vanity string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s&url_type=1", c.apiKey, vanity)
	return c.fetch(ctx, url)
}
