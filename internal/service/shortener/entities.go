package shortener

// removeUserLinksRequest запрос на удаление ссылки из БД
type removeUserLinksRequest struct {
	linkIDs []string
	uid     string
}
