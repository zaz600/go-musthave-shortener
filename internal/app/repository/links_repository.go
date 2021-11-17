package repository

type LinksRepository interface {
	GetURL(idStr string) (string, bool)
	PutURL(longURL string) (int64, error)
	Len() int
}
