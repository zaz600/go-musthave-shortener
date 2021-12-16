package shortener

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type UserLinksResponseEntry struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
