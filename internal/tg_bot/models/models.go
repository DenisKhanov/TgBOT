package models

type ResponseOAuth struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type Message struct {
	Role    string
	Content string
}
