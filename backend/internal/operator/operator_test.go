package operator

import (
	"os"
	"testing"

	"github.com/nats-io/nkeys"
)

func TestLoadFromEnv_Missing(t *testing.T) {
	os.Unsetenv("OPERATOR_SEED")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when OPERATOR_SEED is missing")
	}
}

func TestLoadFromEnv_Invalid(t *testing.T) {
	os.Setenv("OPERATOR_SEED", "not-a-seed")
	defer os.Unsetenv("OPERATOR_SEED")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid seed")
	}
}

func TestLoadFromEnv_Valid(t *testing.T) {
	kp, _ := nkeys.CreateOperator()
	seed, _ := kp.Seed()
	os.Setenv("OPERATOR_SEED", string(seed))
	defer os.Unsetenv("OPERATOR_SEED")

	op, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := op.PublicKey(); len(got) == 0 || got[0] != 'O' {
		t.Fatalf("public key does not start with O: %q", got)
	}
}

func TestSignAccountClaims(t *testing.T) {
	kp, _ := nkeys.CreateOperator()
	seed, _ := kp.Seed()
	os.Setenv("OPERATOR_SEED", string(seed))
	defer os.Unsetenv("OPERATOR_SEED")

	op, _ := LoadFromEnv()

	akp, _ := nkeys.CreateAccount()
	apub, _ := akp.PublicKey()

	claims := NewAccountClaims(apub, "acme")
	tok, err := op.SignAccountClaims(claims)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
}
