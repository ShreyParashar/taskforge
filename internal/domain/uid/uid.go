// Package uid provides a simple UUID type for domain models.
// It wraps a string-based UUID representation so the domain layer
// has no dependency on third-party UUID libraries. The storage layer
// can convert these to/from database UUID types as needed.
package uid

import (
	"crypto/rand"
	"fmt"
)

// ID is a UUID represented as a string.
// Example: "550e8400-e29b-41d4-a716-446655440000"
type ID string

// New generates a new random UUID v4.
func New() ID {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("uid: failed to read random bytes: %v", err))
	}

	// Set version 4 (random) bits
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits (RFC 4122)
	b[8] = (b[8] & 0x3f) | 0x80

	return ID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

// String returns the string representation of the ID.
func (id ID) String() string {
	return string(id)
}

// IsZero returns true if the ID is empty.
func (id ID) IsZero() bool {
	return id == ""
}

// Parse converts a string to an ID.
// It does basic validation on length.
func Parse(s string) (ID, error) {
	if len(s) != 36 {
		return "", fmt.Errorf("uid: invalid length for UUID: %d", len(s))
	}
	return ID(s), nil
}
