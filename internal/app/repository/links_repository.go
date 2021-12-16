package repository

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
)

type LinkEntity struct {
	ID      string `json:"id"`
	LongURL string `json:"long_url"`
	UID     string `json:"uid,omitempty"`
}

type LinksRepository interface {
	Get(linkID string) (string, error)
	Put(link string) (string, error)
	Count() int
}
