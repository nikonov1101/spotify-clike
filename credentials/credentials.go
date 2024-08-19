package credentials

// create new app on Spotify Developer Portal and copy-paste its keys below
// https://developer.spotify.com/documentation/web-api/concepts/apps
// intentionally avoiding flags or env variables here, simply compile credentials into the binary, and forget about it.

func ClientID() string {
	return "your-client-id"
}

func ClientSecret() string {
	return "your-client-secret"
}
