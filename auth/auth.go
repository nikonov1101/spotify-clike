package main

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/nikonov1101/spotify-clike/credentials"
)

const (
	redirectURL = "http://localhost:8000/callback"
)

var scopes = []string{
	"user-read-email",             // whoami?
	"user-read-playback-state",    // playing or not
	"user-read-currently-playing", // now playing what
	"playlist-modify-private",     // add items to playlist
	"user-library-modify",         // unsure but whatever
}

func main() {
	me, err := user.Current()
	if err != nil {
		panic(err)
	}

	state := randSeq(20)
	log.Printf("state = %s", state)
	target := requestTokenURL(state)
	log.Printf("Open the following URL in a browser to authorize the app: %s", target)

	stopc := make(chan struct{})
	startCallbackWebserver(stopc, me.HomeDir)

	<-stopc
	log.Printf("termination")
}

func requestTokenURL(state string) string {
	q := url.Values{}
	q.Add("client_id", credentials.ClientID())
	q.Add("response_type", "code")
	q.Add("scope", strings.Join(scopes, " "))
	q.Add("redirect_uri", redirectURL)
	q.Add("state", state)

	return "https://accounts.spotify.com/authorize?" + q.Encode()
}

func startCallbackWebserver(stopc chan struct{}, homedir string) {
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		errCode := r.URL.Query().Get("error")
		if len(errCode) > 0 {
			panic(errCode)
		}

		if len(state) == 0 {
			panic("no state")
		}

		log.Printf("callback: state=%s; code=%s", state, code)
		exchange(code, homedir)

		w.Write([]byte(`<html><body><h1>Done. It’s now safe to turn off your drum machine.'</h1></body></html>`))
		stopc <- struct{}{}
	})

	go func() {
		log.Printf("start callback webserver")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			panic("listen :8000: " + err.Error())
		}
	}()
}

func exchange(code string, homedir string) {
	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)
	form.Add("redirect_uri", redirectURL)

	req, _ := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(credentials.ClientID(), credentials.ClientSecret())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	writeToken(resp, homedir)
}

func writeToken(resp *http.Response, homedir string) {
	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// make it pretty
	j := map[string]interface{}{}
	json.Unmarshal(bs, &j)
	bs, _ = json.MarshalIndent(j, "", "  ")

	log.Printf("exchange: status %s, body:\n%s", resp.Status, string(bs))
	if resp.StatusCode == http.StatusOK {
		tokenFile := tokenFilePath(homedir)
		log.Printf("saving token to %q ...", tokenFile)
		if err := os.WriteFile(tokenFile, bs, 0600); err != nil {
			panic(err)
		}
	}
}

func tokenFilePath(homedir string) string {
	return filepath.Join(homedir, ".spotify-token.json")
}

func randSeq(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
