package shortener

type removeUserLinksRequest struct {
	linkIDs []string
	uid     string
}
