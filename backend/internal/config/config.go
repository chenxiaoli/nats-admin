package config

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port                   string
	Env                    string
	DatabaseURL            string
	NATSURL                string
	OperatorSeed           string
	SysAccountSeed         string
	MasterKey              []byte
	JWTSecret              []byte
	JWTExpiry              time.Duration
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	required := []string{
		"DATABASE_URL", "NATS_URL", "OPERATOR_SEED",
		"SYSTEM_ACCOUNT_SEED", "MASTER_KEY", "JWT_SECRET",
	}
	for _, k := range required {
		if v.GetString(k) == "" {
			return nil, fmt.Errorf("missing required env: %s", k)
		}
	}

	mkHex := v.GetString("MASTER_KEY")
	mk, err := hex.DecodeString(mkHex)
	if err != nil {
		return nil, fmt.Errorf("MASTER_KEY must be hex: %w", err)
	}
	if len(mk) != 32 {
		return nil, fmt.Errorf("MASTER_KEY must decode to 32 bytes, got %d", len(mk))
	}

	jsHex := v.GetString("JWT_SECRET")
	js, err := hex.DecodeString(jsHex)
	if err != nil {
		return nil, fmt.Errorf("JWT_SECRET must be hex: %w", err)
	}
	if len(js) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(js))
	}

	expiry := v.GetDuration("JWT_EXPIRY")
	if expiry == 0 {
		expiry = 24 * time.Hour
	}

	return &Config{
		Port:                   v.GetString("PORT"),
		Env:                    v.GetString("ENV"),
		DatabaseURL:            v.GetString("DATABASE_URL"),
		NATSURL:                v.GetString("NATS_URL"),
		OperatorSeed:           v.GetString("OPERATOR_SEED"),
		SysAccountSeed:         v.GetString("SYSTEM_ACCOUNT_SEED"),
		MasterKey:              mk,
		JWTSecret:              js,
		JWTExpiry:              expiry,
		BootstrapAdminEmail:    v.GetString("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: v.GetString("BOOTSTRAP_ADMIN_PASSWORD"),
	}, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENV", "development")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	v.SetDefault("BOOTSTRAP_ADMIN_PASSWORD", "changeme")
}
