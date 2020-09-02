package testhelpers

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
	"testing"
)

// RequireRandomPeerID returns a new libp2p peer ID or panics.
func RequireRandomPeerID(t *testing.T) peer.ID {
	pid, err := RandPeerID()
	require.NoError(t, err)
	return pid
}

// MockMessagePoolValidator is a mock validator
type MockMessagePoolValidator struct {
	Valid bool
}

// NewMockMessagePoolValidator creates a MockMessagePoolValidator
func NewMockMessagePoolValidator() *MockMessagePoolValidator {
	return &MockMessagePoolValidator{Valid: true}
}

