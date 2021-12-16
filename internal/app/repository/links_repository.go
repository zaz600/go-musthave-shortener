package repository

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
)

type LinkEntity struct {
	ID      string `json:"id"`
	LongURL string `json:"long_url"`
	User    string `json:"user,omitempty"`
}

type LinksRepository interface {
	Get(linkID string) (string, error)
	Put(link string) (string, error)
	Count() int
}
