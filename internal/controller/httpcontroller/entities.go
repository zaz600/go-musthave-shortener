package httpcontroller

type (
	// ShortenRequest запрос на сокращение ссылки
	ShortenRequest struct {
		// URL ссылка, которую требуется сократить
		URL string `json:"url"`
	}

	// ShortenResponse ответ на запрос на сокращение ссылки
	ShortenResponse struct {
		// Result сокращенная ссылка
		Result string `json:"result"`
		// Error ошибка, если возникла, при сокращении ссылки
		Error string `json:"error,omitempty"`
	}
)

type (
	// UserLinksResponse список ссылок пользователя
	UserLinksResponse []UserLinksResponseEntry

	// UserLinksResponseEntry информация об одной сокращенной ссылке
	UserLinksResponseEntry struct {
		// ShortURL короткая ссылка
		ShortURL string `json:"short_url"`
		// OriginalURL длинная ссылка
		OriginalURL string `json:"original_url"`
	}
)

type (
	// ShortenBatchRequest запрос на сокращение пачки ссылок
	ShortenBatchRequest []ShortenBatchRequestItem

	// ShortenBatchRequestItem элемент в запросе на сокращение пачки ссылок
	ShortenBatchRequestItem struct {
		// URL длинная ссылка, которую надо сократить
		URL string `json:"original_url"`
		// CorrelationID идентификатор ссылки во внешней системе
		CorrelationID string `json:"correlation_id"`
	}

	// ShortenBatchResponse ответ на запрос сокращения пачки ссылок
	ShortenBatchResponse []ShortenBatchResponseItem

	// ShortenBatchResponseItem элемент в ответе на запрос сокращения пачки ссылок
	ShortenBatchResponseItem struct {
		// CorrelationID идентификатор ссылки во внешней системе
		CorrelationID string `json:"correlation_id"`
		// ShortURL сокращенная ссылка
		ShortURL string `json:"short_url"`
	}
)
