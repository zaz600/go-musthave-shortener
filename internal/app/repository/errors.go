package repository

import "fmt"

type LinkExistsError struct {
	LinkID string
	err    error
}

func NewLinkExistsError(linkID string) *LinkExistsError {
	return &LinkExistsError{LinkID: linkID}
}

func (e *LinkExistsError) Error() string {
	return fmt.Sprintf("link alredy exists. short link id: %s", e.LinkID)
}

func (e *LinkExistsError) Unwrap() error {
	return e.err
}
