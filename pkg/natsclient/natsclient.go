// Package natsclient provides a shared NATS JetStream client wrapper.
// Services use this to publish and subscribe to events reliably.
package natsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps a NATS connection and a JetStream context.
type Client struct {
	Conn   *nats.Conn
	JS     jetstream.JetStream
}

// Config holds NATS connection configuration.
type Config struct {
	// URL is the NATS server URL. Example: nats://localhost:4222
	URL string
	// Name is the client name (identifies this service in NATS monitoring).
	Name string
	// MaxReconnects: -1 for infinite reconnects (recommended for services).
	MaxReconnects int
	// ReconnectWait between reconnect attempts.
	ReconnectWait time.Duration
}

func (c *Config) applyDefaults() {
	if c.MaxReconnects == 0 {
		c.MaxReconnects = -1 // infinite
	}
	if c.ReconnectWait == 0 {
		c.ReconnectWait = 2 * time.Second
	}
	if c.URL == "" {
		c.URL = nats.DefaultURL
	}
}

// Connect creates a NATS connection with JetStream enabled.
func Connect(cfg Config) (*Client, error) {
	cfg.applyDefaults()

	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("natsclient: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("natsclient: jetstream init: %w", err)
	}

	return &Client{Conn: nc, JS: js}, nil
}

// MustConnect is like Connect but panics on error.
func MustConnect(cfg Config) *Client {
	c, err := Connect(cfg)
	if err != nil {
		fmt.Printf("FATAL ERROR: %v\n", err); panic(fmt.Sprintf("natsclient: fatal: %v", err))
	}
	return c
}

// Close drains and closes the NATS connection gracefully.
func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Drain() //nolint:errcheck
	}
}

// EnsureStream creates or updates a JetStream stream.
// Safe to call multiple times — idempotent.
func (c *Client) EnsureStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	stream, err := c.JS.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("natsclient: ensure stream %q: %w", cfg.Name, err)
	}
	return stream, nil
}
