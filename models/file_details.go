package models

type FileDetails struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Password   string `json:"password"`
	Notes      string `json:"notes"`
	Username   string `json:"username"`
	TotpSecret string `json:"totp_secret"`
}
