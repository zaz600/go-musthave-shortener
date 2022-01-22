package repository

import (
	"fmt"
)

// LinkExistsError говорит о том, что в хранилище уже есть ссылка,
// которую пытаются сократить повторно.
// Содержит идентификатор короткой ссылки из хранилища
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
