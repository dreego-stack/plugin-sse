package server

import "time"

const (
	defaultReadHeaderTimeout = 10 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
	defaultShutdownTimeout   = 10 * time.Second
)

type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
		ShutdownTimeout:   defaultShutdownTimeout,
	}
}

func (c ServerConfig) withDefaults() ServerConfig {
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = defaultReadTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = defaultWriteTimeout
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = defaultIdleTimeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = defaultMaxHeaderBytes
	}
	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	return c
}

func (a *App) SetServerConfig(c ServerConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.serverConfig = c
	return nil
}
