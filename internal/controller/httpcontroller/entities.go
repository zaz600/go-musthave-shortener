package httpcontroller

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

type ShortenBatchRequest []ShortenBatchRequestItem

type ShortenBatchRequestItem struct {
	URL           string `json:"original_url"`
	CorrelationID string `json:"correlation_id"`
}

type ShortenBatchResponse []ShortenBatchResponseItem

type ShortenBatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
