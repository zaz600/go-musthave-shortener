package repository

type LinksRepository interface {
	Get(linkID string) (string, error)
	Put(link string) (string, error)
	Count() int
}
