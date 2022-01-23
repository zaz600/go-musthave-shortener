package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert.Empty(t, String(-1))
	assert.Empty(t, String(0))
	assert.Len(t, String(1), 1)
	assert.Len(t, String(10), 10)

	assert.NotEqual(t, String(10), String(10))
}

func TestUserID(t *testing.T) {
	assert.Len(t, UserID(), 24)
	assert.NotEqual(t, UserID(), UserID())
}
