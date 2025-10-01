package single

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Host      string
	TCPPort   int
	AuthToken string
	Timeout   time.Duration
	KeepAlive time.Duration
}

type ConfigBuilder struct {
	config Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: Config{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}
}

func (b *ConfigBuilder) WithHost(host string) *ConfigBuilder {
	b.config.Host = strings.TrimSpace(host)
	return b
}

func (b *ConfigBuilder) WithTCPPort(port int) *ConfigBuilder {
	b.config.TCPPort = port
	return b
}

func (b *ConfigBuilder) WithToken(token string) *ConfigBuilder {
	b.config.AuthToken = token
	return b
}

func (b *ConfigBuilder) WithTimeout(d time.Duration) *ConfigBuilder {
	b.config.Timeout = d
	return b
}

func (b *ConfigBuilder) WithKeepAlive(d time.Duration) *ConfigBuilder {
	b.config.KeepAlive = d
	return b
}

func (b *ConfigBuilder) Build() (Config, error) {
	if err := b.validate(); err != nil {
		return Config{}, err
	}
	return b.config, nil
}

func (b *ConfigBuilder) validate() error {
	var errs []error
	if b.config.Host == "" {
		errs = append(errs, errors.New("server address is required"))
	}
	if b.config.TCPPort == 0 {
		errs = append(errs, errors.New("TCP port is required"))
	}
	if b.config.AuthToken == "" {
		errs = append(errs, errors.New("authentication requires a token"))
	}
	return errors.Join(errs...)
}
