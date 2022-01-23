package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkEntity_IsOwnedByUser(t *testing.T) {
	entity := LinkEntity{
		ID:            "",
		OriginalURL:   "",
		UID:           "123",
		CorrelationID: "",
		Removed:       false,
	}
	assert.True(t, entity.IsOwnedByUser("123"))
	assert.False(t, entity.IsOwnedByUser("100500"))

	entity.Removed = true
	assert.True(t, entity.IsOwnedByUser("123"))
	assert.False(t, entity.IsOwnedByUser("100500"))
}

func TestLinkEntity_IsOwnedByUserAndExists(t *testing.T) {
	entity := LinkEntity{
		ID:            "",
		OriginalURL:   "",
		UID:           "123",
		CorrelationID: "",
		Removed:       false,
	}
	assert.True(t, entity.IsOwnedByUserAndExists("123"))
	assert.False(t, entity.IsOwnedByUserAndExists("100500"))

	entity.Removed = true
	assert.False(t, entity.IsOwnedByUserAndExists("123"))
	assert.False(t, entity.IsOwnedByUserAndExists("100500"))
}
