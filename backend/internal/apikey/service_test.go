package apikey

import (
	"strings"
	"testing"
)

func TestHashKey_Deterministic(t *testing.T) {
	a := HashKey("nak_live_abc")
	b := HashKey("nak_live_abc")
	if a != b {
		t.Fatalf("hash not deterministic: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("expected 64-char hex, got %d", len(a))
	}
}

func TestGenerate_Prefix(t *testing.T) {
	raw, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, prefix) {
		t.Fatalf("missing prefix: %s", raw)
	}
	if len(raw) != len(prefix)+randomLen {
		t.Fatalf("wrong length: got %d want %d", len(raw), len(prefix)+randomLen)
	}
}

func TestGenerate_Unique(t *testing.T) {
	a, _ := generate()
	b, _ := generate()
	if a == b {
		t.Fatalf("two generations collided: %s", a)
	}
}
