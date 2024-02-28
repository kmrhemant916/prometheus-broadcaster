package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
    Service struct {
        Port string `yaml:"port"`
    } `yaml:"service"`
	JWTKey string `yaml:"jwt_key"`
    ArangoDB struct {
        Host string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
    } `yaml:"arangodb"`
    SMTP struct {
        Host string `yaml:"host"`
		Port int `yaml:"port"`
    } `yaml:"smtp"`
}

func (c *Config) ReadConf(f string) (*Config, error) {
    buf, err := os.ReadFile(f)
    if err != nil {
        return nil, err
    }
    err = yaml.Unmarshal(buf, c)
    if err != nil {
        return nil, fmt.Errorf("in file %q: %w", f, err)
    }
    return c, err
}