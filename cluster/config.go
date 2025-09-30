package cluster

import (
	"errors"
	"strings"
	"time"
)

type ClusterConfig struct {
	SeedHosts      string // "host1,host2,host3"  (NO port, NO zone, NO shard)
	APIPort        int    // HTTP port for /peers /leader /node-info
	TCPPort        int    // TCP port for framed protocol
	AuthToken      string
	Timeout        time.Duration
	HttpTimeout    time.Duration
	KeepAlive      time.Duration
	MaxActiveConns int // hard cap on open TCP connections
}

type ClusterConfigBuilder struct {
	config ClusterConfig
}

func NewConfigBuilder() *ClusterConfigBuilder {
	return &ClusterConfigBuilder{
		config: ClusterConfig{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}
}

func (b *ClusterConfigBuilder) WithSeedHosts(seed string) *ClusterConfigBuilder {
	b.config.SeedHosts = strings.TrimSpace(seed)
	return b
}

func (b *ClusterConfigBuilder) WithAPIPort(port int) *ClusterConfigBuilder {
	b.config.APIPort = port
	return b
}

func (b *ClusterConfigBuilder) WithTCPPort(port int) *ClusterConfigBuilder {
	b.config.TCPPort = port
	return b
}

func (b *ClusterConfigBuilder) WithToken(token string) *ClusterConfigBuilder {
	b.config.AuthToken = token
	return b
}

func (b *ClusterConfigBuilder) WithTimeout(d time.Duration) *ClusterConfigBuilder {
	b.config.Timeout = d
	return b
}

func (b *ClusterConfigBuilder) WithKeepAlive(d time.Duration) *ClusterConfigBuilder {
	b.config.KeepAlive = d
	return b
}

func (b *ClusterConfigBuilder) Build() (ClusterConfig, error) {
	if err := b.validate(); err != nil {
		return ClusterConfig{}, err
	}
	return b.config, nil
}

func (b *ClusterConfigBuilder) validate() error {
	var errs []error
	if b.config.SeedHosts == "" {
		errs = append(errs, errors.New("at least one seed address is required"))
	}
	if b.config.TCPPort == 0 {
		errs = append(errs, errors.New("TCP port is required"))
	}
	if b.config.APIPort == 0 {
		errs = append(errs, errors.New("API port is required in clustered mode"))
	}
	if b.config.AuthToken == "" {
		errs = append(errs, errors.New("authentication requires a token"))
	}
	return errors.Join(errs...)
}
