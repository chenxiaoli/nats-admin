package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

const operatorName = "platform-operator"

func runBootstrap() error {
	// 1. Operator NKey
	okp, err := nkeys.CreateOperator()
	if err != nil {
		return fmt.Errorf("create operator nkey: %w", err)
	}
	opPub, err := okp.PublicKey()
	if err != nil {
		return fmt.Errorf("operator public key: %w", err)
	}
	opSeed, err := okp.Seed()
	if err != nil {
		return fmt.Errorf("operator seed: %w", err)
	}

	// 2. System Account NKey
	sakp, err := nkeys.CreateAccount()
	if err != nil {
		return fmt.Errorf("create system account nkey: %w", err)
	}
	saPub, err := sakp.PublicKey()
	if err != nil {
		return fmt.Errorf("system account public key: %w", err)
	}
	saSeed, err := sakp.Seed()
	if err != nil {
		return fmt.Errorf("system account seed: %w", err)
	}

	// 3. Operator JWT (signed by Operator)
	opClaims := jwt.NewOperatorClaims(opPub)
	opClaims.Name = operatorName
	opClaims.SystemAccount = saPub
	operatorJWT, err := opClaims.Encode(okp)
	if err != nil {
		return fmt.Errorf("encode operator jwt: %w", err)
	}

	// 4. System Account JWT (signed by Operator)
	saClaims := jwt.NewAccountClaims(saPub)
	saClaims.Name = "SYS"
	systemJWT, err := saClaims.Encode(okp)
	if err != nil {
		return fmt.Errorf("encode system account jwt: %w", err)
	}

	// 5. Write operator.jwt into nats/ dir
	if err := writeOperatorJWT(operatorJWT); err != nil {
		return err
	}

	// 6. Print seeds + public keys to stdout (NOT to file)
	fmt.Println("=== NATS Admin Bootstrap ===")
	fmt.Printf("OPERATOR_NAME=%s\n", operatorName)
	fmt.Printf("OPERATOR_PUBLIC_KEY=%s\n", opPub)
	fmt.Printf("OPERATOR_SEED=%s\n", string(opSeed))
	fmt.Printf("SYSTEM_ACCOUNT_PUBLIC_KEY=%s\n", saPub)
	fmt.Printf("SYSTEM_ACCOUNT_SEED=%s\n", string(saSeed))
	fmt.Println("=== Add the two *SEED lines to backend/.env manually ===")
	fmt.Println("=== operator.jwt written to nats/operator.jwt ===")
	fmt.Println("=== After NATS server starts, push system account JWT to resolver: ===")
	fmt.Printf("=== System Account JWT: %s ===\n", systemJWT)
	return nil
}

func writeOperatorJWT(token string) error {
	dir := "nats"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir nats/: %w", err)
	}
	path := filepath.Join(dir, "operator.jwt")
	out, _ := json.MarshalIndent(map[string]string{"operator": token}, "", "  ")
	return os.WriteFile(path, out, 0o600)
}
