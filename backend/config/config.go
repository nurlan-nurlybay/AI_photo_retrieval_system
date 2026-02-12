package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Version string `yaml:"version"`

		HTTP      HTTP      `yaml:"http"`
		Postgres  Postgres  `yaml:"postgres"`
		Redis     Redis     `yaml:"redis"`
		Seaweedfs Seaweedfs `yaml:"seaweedfs"`
		Clip      Clip      `yaml:"clip"`
		Faiss     Faiss     `yaml:"faiss"`
		Log       Log       `yaml:"log"`
	}

	HTTP struct {
		Host         string        `yaml:"host"`
		Port         uint16        `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"readTimeout"`
		WriteTimeout time.Duration `yaml:"writeTimeout"`
		IdleTimeout  time.Duration `yaml:"idleTimeout"`
	}

	Postgres struct {
		Host              string        `yaml:"host"`
		Port              uint16        `yaml:"port"`
		User              string        `yaml:"user"`
		Password          string        `yaml:"password"`
		DBName            string        `yaml:"dbname"`
		SSLMode           string        `yaml:"sslmode"` // "disable", "require", "verify-full"
		MaxOpenConns      int           `yaml:"maxOpenConns"`
		MaxIdleConns      int           `yaml:"maxIdleConns"`
		ConnMaxLifetime   time.Duration `yaml:"connMaxLifetime"`   // recycle after e.g. 1h
		ConnMaxIdleTime   time.Duration `yaml:"connMaxIdleTime"`   // close idle after e.g. 10m
		HealthCheckPeriod time.Duration `yaml:"healthCheckPeriod"` // ping interval for pgxpool
	}

	Redis struct {
		Addr         string        `yaml:"addr"`     // "host:port"
		Password     string        `yaml:"password"` // optional
		DB           int           `yaml:"db"`       // 0 by default
		DialTimeout  time.Duration `yaml:"dialTimeout"`
		ReadTimeout  time.Duration `yaml:"readTimeout"`
		WriteTimeout time.Duration `yaml:"writeTimeout"`
		PoolSize     int           `yaml:"poolSize"`
	}

	Seaweedfs struct {
		BaseURL   string `yaml:"baseURL"`
		PublicURL string `yaml:"publicURL"` // URL reachable by mobile clients (defaults to BaseURL)
	}

	Clip struct {
		Host string `yaml:"host"`
		Port uint16 `yaml:"port"`
	}

	Faiss struct {
		Host string `yaml:"host"`
		Port uint16 `yaml:"port"`
	}

	Log struct {
		Level        string `yaml:"level"`        // "debug", "info", "warn", "error"
		Format       string `yaml:"format"`       // "text" or "json"
		SourceFolder string `yaml:"sourceFolder"` // project folder name
	}
)

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	d := yaml.NewDecoder(file)
	if err := d.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (p Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.DBName,
		p.SSLMode,
	)
}
