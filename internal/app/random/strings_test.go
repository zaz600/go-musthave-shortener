package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandString(t *testing.T) {
	assert.Empty(t, RandString(-1))
	assert.Empty(t, RandString(0))
	assert.Len(t, RandString(1), 1)
	assert.Len(t, RandString(10), 10)

	assert.NotEqual(t, RandString(10), RandString(10))
}
