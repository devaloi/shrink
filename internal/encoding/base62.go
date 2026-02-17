// Package encoding provides base62 encoding and decoding for URL short codes.
package encoding

import (
	"errors"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const base = int64(len(alphabet))

// ErrInvalidInput is returned when the input cannot be decoded.
var ErrInvalidInput = errors.New("invalid base62 input")

// Encode converts a non-negative integer to a base62 string.
// Returns "a" for 0, "b" for 1, etc.
func Encode(id int64) string {
	if id < 0 {
		return ""
	}
	if id == 0 {
		return string(alphabet[0])
	}

	var sb strings.Builder
	for id > 0 {
		sb.WriteByte(alphabet[id%base])
		id /= base
	}

	// Reverse the string
	result := sb.String()
	runes := []rune(result)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// Decode converts a base62 string back to an integer.
// Returns an error if the string contains invalid characters.
func Decode(code string) (int64, error) {
	if code == "" {
		return 0, ErrInvalidInput
	}

	var result int64
	for _, char := range code {
		idx := strings.IndexRune(alphabet, char)
		if idx == -1 {
			return 0, ErrInvalidInput
		}
		result = result*base + int64(idx)
	}

	return result, nil
}
