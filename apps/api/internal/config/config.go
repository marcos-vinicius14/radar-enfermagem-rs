package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	Port          int
	LogLevel      string
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBMaxConns    int
	DBMinConns    int
	DBConnTimeout time.Duration
}

func Load() (*Config, error) {
	loadDotEnvIfExists()

	cfg := &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		Port:          8080,
		LogLevel:      getEnv("LOG_LEVEL", "debug"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        5432,
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "radar_enfermagem"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		DBMaxConns:    10,
		DBMinConns:    2,
		DBConnTimeout: 5 * time.Second,
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT %q: must be integer: %w", portStr, err)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("PORT %d out of valid range (1-65535)", port)
		}
		cfg.Port = port
	}

	if dbPortStr := os.Getenv("DB_PORT"); dbPortStr != "" {
		dbPort, err := strconv.Atoi(dbPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT %q: must be integer: %w", dbPortStr, err)
		}
		if dbPort < 1 || dbPort > 65535 {
			return nil, fmt.Errorf("DB_PORT %d out of valid range (1-65535)", dbPort)
		}
		cfg.DBPort = dbPort
	}

	if maxConnsStr := os.Getenv("DB_MAX_CONNS"); maxConnsStr != "" {
		maxConns, err := strconv.Atoi(maxConnsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
		}
		cfg.DBMaxConns = maxConns
	}

	if minConnsStr := os.Getenv("DB_MIN_CONNS"); minConnsStr != "" {
		minConns, err := strconv.Atoi(minConnsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MIN_CONNS: %w", err)
		}
		cfg.DBMinConns = minConns
	}

	if timeoutStr := os.Getenv("DB_CONN_TIMEOUT"); timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_CONN_TIMEOUT %q: %w", timeoutStr, err)
		}
		cfg.DBConnTimeout = timeout
	}

	return cfg, nil
}

// DSN gera a URL de conexão do PostgreSQL.
func (c *Config) DSN() string {
	userInfo := url.User(c.DBUser)
	if c.DBPassword != "" {
		userInfo = url.UserPassword(c.DBUser, c.DBPassword)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: fmt.Sprintf("sslmode=%s", c.DBSSLMode),
	}

	return u.String()
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func loadDotEnvIfExists() {
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
			return
		}
	}
}

