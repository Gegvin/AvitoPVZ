package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {

	os.Clearenv()

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "pvz_db", cfg.DBName)
	assert.Equal(t, "your_jwt_secret_key", cfg.JWTSecret)
}

func TestLoadConfig_EnvVars(t *testing.T) {

	t.Setenv("DB_HOST", "testhost")
	t.Setenv("DB_PORT", "5555")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("JWT_SECRET", "testjwtsecret")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "testhost", cfg.DBHost)
	assert.Equal(t, 5555, cfg.DBPort)
	assert.Equal(t, "testuser", cfg.DBUser)
	assert.Equal(t, "testpass", cfg.DBPassword)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "testjwtsecret", cfg.JWTSecret)
}

func TestLoadConfig_InvalidPort(t *testing.T) {

	t.Setenv("DB_PORT", "not-a-number")

	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "pvz_db")
	t.Setenv("JWT_SECRET", "your_jwt_secret_key")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, 5432, cfg.DBPort)
}

func TestGetEnv(t *testing.T) {
	key := "MY_TEST_ENV_VAR"
	fallback := "default_value"

	os.Unsetenv(key)
	value := getEnv(key, fallback)
	assert.Equal(t, fallback, value)

	expectedValue := "actual_value"
	t.Setenv(key, expectedValue)

	value = getEnv(key, fallback)
	assert.Equal(t, expectedValue, value)
}
