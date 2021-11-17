package repository

type LinksRepository interface {
	Get(linkID string) (string, bool)
	Put(link string) (string, error)
	Len() int
}
