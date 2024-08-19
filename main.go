package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gitlab.com/nikonov1101/spotify-clike/credentials"
)

type NowPlaying struct {
	IsPlaying bool `json:"is_playing"`
	Item      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Album struct {
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"album"`
	} `json:"item"`
}

func (p NowPlaying) String() string {
	var artists []string
	for _, a := range p.Item.Album.Artists {
		artists = append(artists, a.Name)
	}
	return fmt.Sprintf("%s: %s (%s) id=%s", strings.Join(artists, ", "),
		p.Item.Name, p.Item.Album.Name, p.Item.ID)
}

type token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	me, err := user.Current()
	if err != nil {
		panic(err)
	}

	defaultpath := filepath.Join(me.HomeDir, ".spotify-token.json")
	tokenPathFlag := flag.String("token", defaultpath, "path to a token file")
	flag.Parse()

	access := newAccessToken(*tokenPathFlag)
	nowPlaying := getCurrentTrack(access)
	likeCurrentTrack(access, nowPlaying.Item.ID)
	fmt.Println("[+]", nowPlaying.String())
}

func newAccessToken(tokenPath string) string {
	bs, err := os.ReadFile(tokenPath)
	if err != nil {
		panic(err)
	}

	tok := token{}
	if err := json.Unmarshal(bs, &tok); err != nil {
		panic(err)
	}

	if tok.RefreshToken == "" {
		panic("no refresh_token in a file")
	}

	access, err := refreshAccessToken(tok.RefreshToken)
	if err != nil {
		panic(err)
	}

	return access
}

func refreshAccessToken(refreshToken string) (string, error) {
	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("refresh_token", refreshToken)

	req, _ := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(credentials.ClientID(), credentials.ClientSecret())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("refresh: status: %s; body: %s", resp.Status, string(bs))
		return "", errors.New("non-200 response: " + resp.Status)
	}

	newToken := token{}
	if err := json.Unmarshal(bs, &newToken); err != nil {
		return "", fmt.Errorf("unmarshal token: %w", err)
	}

	return newToken.AccessToken, nil
}

func getCurrentTrack(act string) NowPlaying {
	req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/player/currently-playing", nil)
	req.Header.Add("Authorization", "Bearer "+act)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}

	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	p := NowPlaying{}
	if err := json.Unmarshal(bs, &p); err != nil {
		panic(err)
	}

	return p
}

func likeCurrentTrack(act string, id string) {
	req, _ := http.NewRequest(http.MethodPut, "https://api.spotify.com/v1/me/tracks?ids="+id, nil)
	req.Header.Add("Authorization", "Bearer "+act)
	req.Header.Add("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}
}
