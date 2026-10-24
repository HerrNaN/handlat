package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateUsername(t *testing.T) {
	t.Parallel()

	username, err := generateUsername()

	require.NoError(t, err)

	parts := strings.Split(username, "-")
	require.Len(t, parts, 2)
	assert.Contains(t, usernameAdjectives[:], parts[0])
	assert.Contains(t, usernameAnimals[:], parts[1])
}
