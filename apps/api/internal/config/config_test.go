package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars
	envVars := []string{
		"APP_ENV", "PORT", "LOG_LEVEL",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"DB_SSLMODE", "DB_MAX_CONNS", "DB_MIN_CONNS", "DB_CONN_TIMEOUT",
		"ENABLE_SCHEDULER", "COLLECTOR_CRON_SCHEDULE", "COLLECTOR_CONCURRENCY",
		"COLLECTOR_RATE_LIMIT_RPS", "COLLECTOR_RATE_LIMIT_BURST",
		"COLLECTOR_STATUS_UNKNOWN_HOURS", "COLLECTOR_STATUS_EXPIRED_HOURS",
		"COLLECTOR_RUN_ON_STARTUP",
	}
	for _, env := range envVars {
		_ = os.Unsetenv(env)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error loading defaults, got: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv 'development', got: %s", cfg.AppEnv)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got: %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %s", cfg.LogLevel)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected DBHost 'localhost', got: %s", cfg.DBHost)
	}
	if cfg.DBPort != 5432 {
		t.Errorf("expected DBPort 5432, got: %d", cfg.DBPort)
	}
	if cfg.DBUser != "postgres" {
		t.Errorf("expected DBUser 'postgres', got: %s", cfg.DBUser)
	}
	if cfg.DBName != "radar_enfermagem" {
		t.Errorf("expected DBName 'radar_enfermagem', got: %s", cfg.DBName)
	}
	if cfg.DBSSLMode != "disable" {
		t.Errorf("expected DBSSLMode 'disable', got: %s", cfg.DBSSLMode)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("expected DBMaxConns 10, got: %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 2 {
		t.Errorf("expected DBMinConns 2, got: %d", cfg.DBMinConns)
	}
	if cfg.DBConnTimeout != 5*time.Second {
		t.Errorf("expected DBConnTimeout 5s, got: %v", cfg.DBConnTimeout)
	}
	if cfg.EnableScheduler != false {
		t.Errorf("expected EnableScheduler false, got: %v", cfg.EnableScheduler)
	}
	if cfg.CollectorCronSchedule != "0 */2 * * *" {
		t.Errorf("expected CollectorCronSchedule '0 */2 * * *', got: %s", cfg.CollectorCronSchedule)
	}
	if cfg.CollectorConcurrency != 3 {
		t.Errorf("expected CollectorConcurrency 3, got: %d", cfg.CollectorConcurrency)
	}
	if cfg.CollectorRateLimitRPS != 3.0 {
		t.Errorf("expected CollectorRateLimitRPS 3.0, got: %v", cfg.CollectorRateLimitRPS)
	}
	if cfg.CollectorRateLimitBurst != 5 {
		t.Errorf("expected CollectorRateLimitBurst 5, got: %d", cfg.CollectorRateLimitBurst)
	}
	if cfg.CollectorStatusUnknownHours != 24 {
		t.Errorf("expected CollectorStatusUnknownHours 24, got: %d", cfg.CollectorStatusUnknownHours)
	}
	if cfg.CollectorStatusExpiredHours != 168 {
		t.Errorf("expected CollectorStatusExpiredHours 168, got: %d", cfg.CollectorStatusExpiredHours)
	}
	if cfg.CollectorRunOnStartup != false {
		t.Errorf("expected CollectorRunOnStartup false, got: %v", cfg.CollectorRunOnStartup)
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PORT", "9000")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "custom_user")
	t.Setenv("DB_PASSWORD", "custom_pass")
	t.Setenv("DB_NAME", "custom_db")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_MAX_CONNS", "20")
	t.Setenv("DB_MIN_CONNS", "5")
	t.Setenv("DB_CONN_TIMEOUT", "10s")
	t.Setenv("ENABLE_SCHEDULER", "true")
	t.Setenv("COLLECTOR_CRON_SCHEDULE", "0 */4 * * *")
	t.Setenv("COLLECTOR_CONCURRENCY", "5")
	t.Setenv("COLLECTOR_RATE_LIMIT_RPS", "5.5")
	t.Setenv("COLLECTOR_RATE_LIMIT_BURST", "10")
	t.Setenv("COLLECTOR_STATUS_UNKNOWN_HOURS", "48")
	t.Setenv("COLLECTOR_STATUS_EXPIRED_HOURS", "336")
	t.Setenv("COLLECTOR_RUN_ON_STARTUP", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error loading custom env, got: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv 'production', got: %s", cfg.AppEnv)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected Port 9000, got: %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got: %s", cfg.LogLevel)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("expected DBHost 'db.example.com', got: %s", cfg.DBHost)
	}
	if cfg.DBPort != 5433 {
		t.Errorf("expected DBPort 5433, got: %d", cfg.DBPort)
	}
	if cfg.DBUser != "custom_user" {
		t.Errorf("expected DBUser 'custom_user', got: %s", cfg.DBUser)
	}
	if cfg.DBPassword != "custom_pass" {
		t.Errorf("expected DBPassword 'custom_pass', got: %s", cfg.DBPassword)
	}
	if cfg.DBName != "custom_db" {
		t.Errorf("expected DBName 'custom_db', got: %s", cfg.DBName)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("expected DBSSLMode 'require', got: %s", cfg.DBSSLMode)
	}
	if cfg.DBMaxConns != 20 {
		t.Errorf("expected DBMaxConns 20, got: %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 5 {
		t.Errorf("expected DBMinConns 5, got: %d", cfg.DBMinConns)
	}
	if cfg.DBConnTimeout != 10*time.Second {
		t.Errorf("expected DBConnTimeout 10s, got: %v", cfg.DBConnTimeout)
	}
	if cfg.EnableScheduler != true {
		t.Errorf("expected EnableScheduler true, got: %v", cfg.EnableScheduler)
	}
	if cfg.CollectorCronSchedule != "0 */4 * * *" {
		t.Errorf("expected CollectorCronSchedule '0 */4 * * *', got: %s", cfg.CollectorCronSchedule)
	}
	if cfg.CollectorConcurrency != 5 {
		t.Errorf("expected CollectorConcurrency 5, got: %d", cfg.CollectorConcurrency)
	}
	if cfg.CollectorRateLimitRPS != 5.5 {
		t.Errorf("expected CollectorRateLimitRPS 5.5, got: %v", cfg.CollectorRateLimitRPS)
	}
	if cfg.CollectorRateLimitBurst != 10 {
		t.Errorf("expected CollectorRateLimitBurst 10, got: %d", cfg.CollectorRateLimitBurst)
	}
	if cfg.CollectorStatusUnknownHours != 48 {
		t.Errorf("expected CollectorStatusUnknownHours 48, got: %d", cfg.CollectorStatusUnknownHours)
	}
	if cfg.CollectorStatusExpiredHours != 336 {
		t.Errorf("expected CollectorStatusExpiredHours 336, got: %d", cfg.CollectorStatusExpiredHours)
	}
	if cfg.CollectorRunOnStartup != true {
		t.Errorf("expected CollectorRunOnStartup true, got: %v", cfg.CollectorRunOnStartup)
	}

}

func TestLoad_EdgeCasesAndValidation(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVal  string
		wantErr bool
	}{
		{
			name:    "port non numeric",
			envKey:  "PORT",
			envVal:  "invalid_port",
			wantErr: true,
		},
		{
			name:    "port below 1",
			envKey:  "PORT",
			envVal:  "0",
			wantErr: true,
		},
		{
			name:    "port above 65535",
			envKey:  "PORT",
			envVal:  "70000",
			wantErr: true,
		},
		{
			name:    "db port non numeric",
			envKey:  "DB_PORT",
			envVal:  "not_a_number",
			wantErr: true,
		},
		{
			name:    "db timeout invalid duration",
			envKey:  "DB_CONN_TIMEOUT",
			envVal:  "not-a-duration",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)
			_, err := config.Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("config.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDSN(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "radar")
	t.Setenv("DB_SSLMODE", "disable")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	want := "postgres://user:secret@localhost:5432/radar?sslmode=disable"
	got := cfg.DSN()
	if got != want {
		t.Errorf("expected DSN %q, got %q", want, got)
	}
}
