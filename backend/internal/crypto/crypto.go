package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(masterKey, plaintext []byte) (ciphertext, nonce []byte, err error) {
	if len(masterKey) != 32 {
		return nil, nil, fmt.Errorf("master key must be 32 bytes, got %d", len(masterKey))
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return
}

func Decrypt(masterKey, ciphertext, nonce []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes, got %d", len(masterKey))
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("bad nonce size: %d", len(nonce))
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
