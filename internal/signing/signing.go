// Package signing provides a simple utility for computing HMAC-SHA256
// digests.
//
// The main entry point is [CalcHash], which returns the digest as
// a hexadecimal string.
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HeaderName is the name of the HTTP header used to transmit
//
// Example header in an HTTP request:
//
//	HashSHA256: 5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843
const HeaderName = "HashSHA256"

// CalcHash computes an HMAC-SHA256 digest of data using the provided key
// and returns it as a hexadecimal string.
//
// The returned string is always 64 hexadecimal characters long
// (256 bits) when key is non-empty.
func CalcHash(data, key []byte) string {
	if len(key) == 0 {
		return ""
	}

	h := hmac.New(sha256.New, key)
	h.Write(data)

	return hex.EncodeToString(h.Sum(nil))
}
