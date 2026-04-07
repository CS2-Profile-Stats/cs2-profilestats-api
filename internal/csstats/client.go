package csstats

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
)

type Profile struct {
	Name  *string `json:"name"`
	Stats *Stats  `json:"stats"`
}

type Stats struct {
	KDRatio       *float64  `json:"kd_ratio"`
	HLTVRating    *float64  `json:"hltv_rating"`
	Matches       *int      `json:"matches"`
	WinRate       *int      `json:"win_rate"`
	HSPercentage  *int      `json:"hs_percentage"`
	ADR           *int      `json:"adr"`
	Clutch        *int      `json:"clutch"`
	RecentResults []*string `json:"recent_results"`
	MostPlayedMap *string   `json:"most_played_map"`
}

type Client struct {
	allocCtx context.Context
	cancel   context.CancelFunc
	tabs     chan struct{}
}

func NewClient() *Client {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-infobars", true),
		chromedp.UserAgent(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
				"AppleWebKit/537.36 (KHTML, like Gecko) "+
				"Chrome/124.0.0.0 Safari/537.36",
		),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &Client{
		allocCtx: allocCtx,
		cancel:   cancel,
		tabs:     make(chan struct{}, 7),
	}
}

func (c *Client) Close() {
	c.cancel()
}

func (c *Client) GetProfile(ctx context.Context, steamID string) (*Profile, error) {
	select {
	case c.tabs <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	defer func() { <-c.tabs }()
	tabCtx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()

	tabCtx, cancel = context.WithTimeout(tabCtx, 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://csstats.gg/player/%s", steamID)

	var raw map[string]any

	err := chromedp.Run(tabCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Evaluate(
				`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`,
				nil,
			).Do(ctx)
		}),

		chromedp.Navigate(url),

		// wait for overview to be visible first
		chromedp.WaitVisible(`#player-overview`, chromedp.ByQuery),

		// csstats css is horrible
		chromedp.Evaluate(`
			(function() {
				function getStatPanel(heading) {
					var panels = document.querySelectorAll('.stat-panel');
					for (var p of panels) {
						var h = p.querySelector('.stat-heading');
						if (h && h.innerText.trim() === heading) return p;
					}
					return null;
				}
				function getUnnamedStatPerc(heading) {
					var p = getStatPanel(heading);
					if (!p) return '';
					var m = p.innerText.match(/(\d+)%/);
					return m ? m[1] : '';
				}
				function getUnnamedStatNum(heading) {
					var p = getStatPanel(heading);
					if (!p) return '';
					var m = p.innerText.match(/(\d+)/);
					return m ? m[1] : '';
				}

				var dots = document.querySelectorAll('.match-dot');
				var recentResults = [];
				for (var i = 0; i < Math.min(5, dots.length); i++) {
					var d = dots[i];
					if (d.classList.contains('match-win')) recentResults.push('W');
					else if (d.classList.contains('match-lose')) recentResults.push('L');
					else if (d.classList.contains('match-draw')) recentResults.push('T');
				}

				var mostPlayedPanel = getStatPanel('Most Played');
				var mostPlayedSpan = mostPlayedPanel ? mostPlayedPanel.querySelector('span') : null;

				var matchesNode = document.evaluate(
					'//div[contains(@class,"total-stat")][.//span[contains(@class,"total-label") and normalize-space()="Played"]]//span[contains(@class,"total-value")]',
					document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null
				);

				return {
					name:          (document.querySelector('#player-name') || {innerText:''}).innerText.trim(),
					kdRatio:       (document.querySelector('#kpd span') || {innerText:''}).innerText.trim(),
					hltvRating:    (document.querySelector('#rating span') || {innerText:''}).innerText.trim(),
					matches:       matchesNode.singleNodeValue ? matchesNode.singleNodeValue.innerText.trim() : '',
					winRate:       getUnnamedStatPerc('WIN RATE'),
					hsPercentage:  getUnnamedStatPerc('HS%'),
					adr:           getUnnamedStatNum('ADR'),
					clutch:        getUnnamedStatPerc('CLUTCH SUCCESS'),
					mostPlayedMap: mostPlayedSpan ? mostPlayedSpan.innerText.trim() : '',
					recentResults: recentResults,
				};
			})()
		`, &raw),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to scrape CSStats profile for %s: %w", steamID, err)
	}

	name := parseString(raw["name"])

	kdRatio := parseFloat(raw["kdRatio"])
	hltvRating := parseFloat(raw["hltvRating"])
	matches := parseInt(raw["matches"])
	winRate := parseInt(raw["winRate"])
	hsPercentage := parseInt(raw["hsPercentage"])
	adr := parseInt(raw["adr"])
	clutch := parseInt(raw["clutch"])
	mostPlayedMap := parseString(raw["mostPlayedMap"])

	var recentResults []*string
	if arr, ok := raw["recentResults"].([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				s := s
				recentResults = append(recentResults, &s)
			}
		}
	}

	return &Profile{
		Name: name,
		Stats: &Stats{
			KDRatio:       kdRatio,
			HLTVRating:    hltvRating,
			Matches:       matches,
			WinRate:       winRate,
			HSPercentage:  hsPercentage,
			ADR:           adr,
			Clutch:        clutch,
			RecentResults: recentResults,
			MostPlayedMap: mostPlayedMap,
		},
	}, nil
}

func parseString(v any) *string {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return &s
}

func parseFloat(v any) *float64 {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

func parseInt(v any) *int {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &i
}
