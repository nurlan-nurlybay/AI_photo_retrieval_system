package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Version string `yaml:"version"`

		HTTP     HTTP     `yaml:"http"`
		Postgres Postgres `yaml:"postgres"`
		Log      Log      `yaml:"log"`
	}

	HTTP struct {
		Host         string        `yaml:"host"`
		Port         uint16        `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"readTimeout"`
		WriteTimeout time.Duration `yaml:"writeTimeout"`
		IdleTimeout  time.Duration `yaml:"idleTimeout"`
	}

	Postgres struct {
		Host            string        `yaml:"host"`
		Port            uint16        `yaml:"port"`
		User            string        `yaml:"user"`
		Password        string        `yaml:"password"`
		DBName          string        `yaml:"dbname"`
		MaxOpenConns    int           `yaml:"maxOpenConns"`
		MaxIdleConns    int           `yaml:"maxIdleConns"`
		ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
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
