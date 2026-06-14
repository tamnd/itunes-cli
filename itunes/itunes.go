// Package itunes is the library behind the itunes command line:
// the HTTP client, request shaping, and the typed data models for the
// iTunes Search API (itunes.apple.com). No API key required.
package itunes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Host is the API host this client talks to.
const Host = "itunes.apple.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://itunes.apple.com",
		UserAgent: "itunes-cli/0.1.0 (github.com/tamnd/itunes-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the iTunes API over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Result is one item returned by the iTunes API.
type Result struct {
	WrapperType    string  `json:"wrapperType"`
	Kind           string  `json:"kind,omitempty"`
	CollectionID   int64   `json:"collectionId,omitempty"`
	TrackID        int64   `json:"trackId,omitempty"`
	ArtistID       int64   `json:"artistId,omitempty"`
	ArtistName     string  `json:"artistName,omitempty"`
	CollectionName string  `json:"collectionName,omitempty"`
	TrackName      string  `json:"trackName,omitempty"`
	PreviewURL     string  `json:"previewUrl,omitempty"`
	ArtworkURL     string  `json:"artworkUrl100,omitempty"`
	Price          float64 `json:"trackPrice,omitempty"`
	Currency       string  `json:"currency,omitempty"`
	ReleaseDate    string  `json:"releaseDate,omitempty"`
	Genre          string  `json:"primaryGenreName,omitempty"`
	Country        string  `json:"country,omitempty"`
	TrackCount     int     `json:"trackCount,omitempty"`
	FeedURL        string  `json:"feedUrl,omitempty"`
}

// searchResponse is the wire format for /search and /lookup.
type searchResponse struct {
	ResultCount int      `json:"resultCount"`
	Results     []Result `json:"results"`
}

// Search searches the iTunes catalog.
// entity is one of: song, album, musicArtist, movie, tvShow, podcast, software.
func (c *Client) Search(ctx context.Context, term, entity, country string, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = 25
	}
	if country == "" {
		country = "US"
	}
	if entity == "" {
		entity = "song"
	}
	u := fmt.Sprintf("%s/search?term=%s&entity=%s&limit=%d&country=%s",
		c.cfg.BaseURL,
		url.QueryEscape(term),
		url.QueryEscape(entity),
		limit,
		url.QueryEscape(country),
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	return resp.Results, nil
}

// Lookup looks up an item by its Apple ID, optionally filtering by entity.
func (c *Client) Lookup(ctx context.Context, id int64, entity string) ([]Result, error) {
	u := fmt.Sprintf("%s/lookup?id=%s", c.cfg.BaseURL, strconv.FormatInt(id, 10))
	if entity != "" {
		u += "&entity=" + url.QueryEscape(entity)
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode lookup: %w", err)
	}
	return resp.Results, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
