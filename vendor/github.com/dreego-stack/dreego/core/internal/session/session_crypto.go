package session

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"io"
)

const encMarker byte = 1

func encryptPayload(key, plaintext []byte, r io.Reader) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(r, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.RawURLEncoding.EncodeToString(ciphertext)), nil
}

func decryptPayload(key, ciphertext []byte) ([]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, false
	}
	if len(raw) < gcm.NonceSize() {
		return nil, false
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, false
	}
	return pt, true
}
