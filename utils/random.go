package utils

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// RandomString returns a string of length len consisting of random characters (alphanumeric).
func RandomString(len int) string {
	b := make([]byte, (len+1)/2)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		// reading from crypto/rand.Reader should never fail
		panic(err)
	}
	return hex.EncodeToString(b)[:len]
}
