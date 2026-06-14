// Package itunes exposes the iTunes Search API as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/itunes-cli/itunes"
//
// The same Domain also builds the standalone itunes binary (see cli.NewApp).
package itunes

import (
	"context"
	"fmt"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the itunes driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "itunes",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "itunes",
			Short:  "Search the iTunes catalog for songs, albums, artists, podcasts and more",
			Long: `itunes searches the free iTunes Search API (itunes.apple.com).
No API key required. Search for songs, albums, artists, movies, podcasts,
and software by term or look up any item by Apple ID.`,
			Site: Host,
			Repo: "https://github.com/tamnd/itunes-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// search: general search
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search the iTunes catalog by term",
	}, searchOp)

	// lookup: lookup by Apple ID
	kit.Handle(app, kit.OpMeta{
		Name:    "lookup",
		Group:   "read",
		List:    true,
		Summary: "Look up an item by its Apple ID",
	}, lookupOp)

	// artist: shortcut for --type musicArtist
	kit.Handle(app, kit.OpMeta{
		Name:    "artist",
		Group:   "read",
		List:    true,
		Summary: "Search for artists by name",
	}, artistOp)

	// album: shortcut for --type album
	kit.Handle(app, kit.OpMeta{
		Name:    "album",
		Group:   "read",
		List:    true,
		Summary: "Search for albums by name",
	}, albumOp)

	// podcast: shortcut for --type podcast
	kit.Handle(app, kit.OpMeta{
		Name:    "podcast",
		Group:   "read",
		List:    true,
		Summary: "Search for podcasts by name",
	}, podcastOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type searchInput struct {
	Term    string  `kit:"flag" help:"search term"`
	Type    string  `kit:"flag" help:"entity type: song, album, musicArtist, movie, tvShow, podcast, software"`
	Limit   int     `kit:"flag" help:"max results (default 25)"`
	Country string  `kit:"flag" help:"two-letter country code (default US)"`
	Client  *Client `kit:"inject"`
}

type lookupInput struct {
	ID     int64   `kit:"flag" help:"Apple item ID"`
	Type   string  `kit:"flag" help:"entity type filter"`
	Client *Client `kit:"inject"`
}

type shortcutInput struct {
	Term    string  `kit:"flag" help:"search term"`
	Limit   int     `kit:"flag" help:"max results (default 25)"`
	Country string  `kit:"flag" help:"two-letter country code (default US)"`
	Client  *Client `kit:"inject"`
}

// --- handlers ---

func searchOp(ctx context.Context, in searchInput, emit func(Result) error) error {
	results, err := in.Client.Search(ctx, in.Term, in.Type, in.Country, in.Limit)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func lookupOp(ctx context.Context, in lookupInput, emit func(Result) error) error {
	results, err := in.Client.Lookup(ctx, in.ID, in.Type)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func artistOp(ctx context.Context, in shortcutInput, emit func(Result) error) error {
	results, err := in.Client.Search(ctx, in.Term, "musicArtist", in.Country, in.Limit)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func albumOp(ctx context.Context, in shortcutInput, emit func(Result) error) error {
	results, err := in.Client.Search(ctx, in.Term, "album", in.Country, in.Limit)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func podcastOp(ctx context.Context, in shortcutInput, emit func(Result) error) error {
	results, err := in.Client.Search(ctx, in.Term, "podcast", in.Country, in.Limit)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty itunes reference")
	}
	return "result", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "result":
		return fmt.Sprintf("https://%s/lookup?id=%s", Host, id), nil
	default:
		return "", errs.Usage("itunes has no resource type %q", uriType)
	}
}
