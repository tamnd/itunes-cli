package itunes_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/itunes-cli/itunes"
)

const searchJSON = `{
  "resultCount": 2,
  "results": [
    {
      "wrapperType": "track",
      "kind": "song",
      "trackId": 159292912,
      "artistId": 159292911,
      "artistName": "Taylor Swift",
      "trackName": "Love Story",
      "collectionName": "Fearless",
      "artworkUrl100": "https://is1-ssl.mzstatic.com/image/thumb/Music/v4/1.jpg",
      "trackPrice": 1.29,
      "currency": "USD",
      "releaseDate": "2008-11-11T00:00:00Z",
      "primaryGenreName": "Country"
    },
    {
      "wrapperType": "track",
      "kind": "song",
      "trackId": 159292913,
      "artistName": "Taylor Swift",
      "trackName": "You Belong with Me",
      "collectionName": "Fearless",
      "trackPrice": 1.29,
      "currency": "USD",
      "primaryGenreName": "Country"
    }
  ]
}`

const lookupJSON = `{
  "resultCount": 1,
  "results": [
    {
      "wrapperType": "collection",
      "collectionId": 401130553,
      "artistName": "The Beatles",
      "collectionName": "Abbey Road",
      "trackCount": 17,
      "releaseDate": "1969-09-26T00:00:00Z",
      "primaryGenreName": "Rock",
      "country": "USA"
    }
  ]
}`

func newTestClient(srv *httptest.Server) *itunes.Client {
	cfg := itunes.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return itunes.NewClient(cfg)
}

func TestSearchParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("term") == "" {
			t.Error("term param missing")
		}
		_, _ = fmt.Fprint(w, searchJSON)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	results, err := c.Search(context.Background(), "taylor swift", "song", "US", 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].ArtistName != "Taylor Swift" {
		t.Errorf("ArtistName = %q, want Taylor Swift", results[0].ArtistName)
	}
	if results[0].TrackName != "Love Story" {
		t.Errorf("TrackName = %q, want Love Story", results[0].TrackName)
	}
	if results[0].Price != 1.29 {
		t.Errorf("Price = %v, want 1.29", results[0].Price)
	}
}

func TestLookupParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookup" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "401130553" {
			t.Errorf("id param = %q, want 401130553", r.URL.Query().Get("id"))
		}
		_, _ = fmt.Fprint(w, lookupJSON)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	results, err := c.Lookup(context.Background(), 401130553, "album")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].CollectionName != "Abbey Road" {
		t.Errorf("CollectionName = %q, want Abbey Road", results[0].CollectionName)
	}
	if results[0].TrackCount != 17 {
		t.Errorf("TrackCount = %d, want 17", results[0].TrackCount)
	}
}

func TestSearchEntityParam(t *testing.T) {
	var gotEntity string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEntity = r.URL.Query().Get("entity")
		_, _ = fmt.Fprint(w, `{"resultCount":0,"results":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Search(context.Background(), "beatles", "musicArtist", "US", 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotEntity != "musicArtist" {
		t.Errorf("entity = %q, want musicArtist", gotEntity)
	}
}

func TestSearchDefaultLimit(t *testing.T) {
	var gotLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLimit = r.URL.Query().Get("limit")
		_, _ = fmt.Fprint(w, `{"resultCount":0,"results":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Search(context.Background(), "serial", "podcast", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotLimit != "25" {
		t.Errorf("limit = %q, want 25", gotLimit)
	}
}

func TestUserAgentSent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, `{"resultCount":0,"results":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _ = c.Search(context.Background(), "test", "song", "US", 1)
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, `{"resultCount":0,"results":[]}`)
	}))
	defer srv.Close()

	cfg := itunes.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := itunes.NewClient(cfg)

	_, err := c.Search(context.Background(), "test", "song", "US", 1)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
