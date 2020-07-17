package views

type AuthRequest struct {
	GUID string `json:"guid"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type DeleteTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
