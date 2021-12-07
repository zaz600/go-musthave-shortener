package repository

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
)

type LinksRepository interface {
	Get(linkID string) (string, error)
	Put(link string) (string, error)
	Count() int
}
