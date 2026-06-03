package crypto

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("SOABCDEFGHIJKLMNOPQRSTUVWXYZ234567")

	ct, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}
	got, err := Decrypt(key, ct, nonce)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestBadKey(t *testing.T) {
	if _, _, err := Encrypt(make([]byte, 16), []byte("x")); err == nil {
		t.Fatal("expected error for 16-byte key")
	}
}

func TestTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	ct, nonce, _ := Encrypt(key, []byte("hello"))
	ct[0] ^= 0x01
	if _, err := Decrypt(key, ct, nonce); err == nil {
		t.Fatal("expected decrypt failure for tampered ciphertext")
	}
}

func TestDistinctNonces(t *testing.T) {
	key := make([]byte, 32)
	_, n1, _ := Encrypt(key, []byte("a"))
	_, n2, _ := Encrypt(key, []byte("a"))
	if bytes.Equal(n1, n2) {
		t.Fatal("nonces collided (very unlikely with crypto/rand)")
	}
}
