package repository

type LinksRepository interface {
	Get(linkID string) (string, bool)
	Put(link string) (int64, error)
	Len() int
}
