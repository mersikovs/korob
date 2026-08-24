package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const HeaderName = "HashSHA256"

func CalcHash(data, key []byte) string {
	if len(key) == 0 {
		return ""
	}

	h := hmac.New(sha256.New, key)
	h.Write(data)

	return hex.EncodeToString(h.Sum(nil))
}
