package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "plainpassword"
	hash, err := HashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "plainpassword"
	hash, _ := HashPassword(password)

	assert.True(t, CheckPasswordHash(password, hash))

	assert.False(t, CheckPasswordHash("wrongpassword", hash))

	assert.False(t, CheckPasswordHash(password, "invalidhash"))
}
