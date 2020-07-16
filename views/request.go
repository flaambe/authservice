package views

type AuthRequest struct {
	GUID string `json:"guid"`
}

type RefreshTokenRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type DeleteTokenRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type DeleteAllTokensRequest struct {
	AccessToken string `json:"access_token"`
}
