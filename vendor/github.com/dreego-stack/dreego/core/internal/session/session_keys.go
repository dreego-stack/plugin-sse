package session

import (
	"crypto/hmac"
	"crypto/sha256"
)

type sessionKeys struct {
	sig []byte
	enc []byte
}

func deriveKeys(secret []byte) sessionKeys {
	sig := hmac.New(sha256.New, secret)
	sig.Write([]byte("dreego-session-sig"))

	enc := hmac.New(sha256.New, secret)
	enc.Write([]byte("dreego-session-enc"))

	return sessionKeys{sig: sig.Sum(nil), enc: enc.Sum(nil)}
}
