package temporal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DataDog/temporalite"
	"github.com/go-logr/logr"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/temporal"
)

type Client struct {
	Embedded  bool
	Ephemeral bool
	Namespace string
	Server    *temporalite.Server
	Client    client.Client
}

func New() *Client {
	return &Client{}
}

func (c *Client) Create(logger logr.Logger) (err error) {
	if c.Namespace == "" {
		return errors.New("namespace is undefined")
	}

	opts := client.Options{
		Namespace: c.Namespace,
		Logger:    clientLogger{logger},
	}

	if c.Embedded {
		if err := c.embedTemporal(logger.WithName("server")); err != nil {
			return fmt.Errorf("error starting temporalite: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if c.Client, err = c.Server.NewClientWithOptions(ctx, opts); err != nil {
			return err
		}
	} else if c.Client, err = client.NewClient(opts); err != nil {
		return err
	}

	return nil
}

func (c *Client) Close() error {
	if c.Client != nil {
		c.Client.Close()
	}

	if c.Server != nil {
		c.Server.Stop()
	}

	return nil
}

func (c *Client) embedTemporal(logger logr.Logger) (err error) {
	opts := []temporalite.ServerOption{
		temporalite.WithNamespaces(c.Namespace),
		temporalite.WithDynamicPorts(),
		temporalite.WithUpstreamOptions(
			temporal.WithLogger(serverLogger{logger}),
		),
	}
	if c.Ephemeral {
		opts = append(opts, temporalite.WithPersistenceDisabled())
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		opts = append(opts, temporalite.WithDatabaseFilePath(
			filepath.Join(configDir, "temporalite.db"),
		))
	}

	c.Server, err = temporalite.NewServer(opts...)
	if err != nil {
		return err
	}

	go c.Server.Start()

	return nil
}

// clientLogger wraps the application logger for compatibility with Temporal.
type clientLogger struct {
	logger logr.Logger
}

func (l clientLogger) Debug(msg string, keyvals ...interface{}) {
	l.logger.V(8).Info(msg, keyvals...)
}

func (l clientLogger) Info(msg string, keyvals ...interface{}) {
	l.logger.V(7).Info(msg, keyvals...)
}

func (l clientLogger) Warn(msg string, keyvals ...interface{}) {
	l.logger.V(6).Info(msg, keyvals...)
}

func (l clientLogger) Error(msg string, keyvals ...interface{}) {
	l.logger.V(5).Info(msg, keyvals...)
}

// serverLogger wraps the application logger for compatibility with Temporal.
type serverLogger struct {
	logger logr.Logger
}

func (l serverLogger) kv(msg string, tags []tag.Tag) []interface{} {
	keyvals := []interface{}{}
	for _, tag := range tags {
		keyvals = append(keyvals, []interface{}{tag.Key(), tag.Value()}...)
	}
	return keyvals
}

func (l serverLogger) Debug(msg string, tags ...tag.Tag) {
	l.logger.V(8).Info(msg, l.kv(msg, tags)...)
}

func (l serverLogger) Info(msg string, tags ...tag.Tag) {
	l.logger.V(8).Info(msg, l.kv(msg, tags)...)
}

func (l serverLogger) Warn(msg string, tags ...tag.Tag) {
	l.logger.V(8).Info(msg, l.kv(msg, tags)...)
}

func (l serverLogger) Error(msg string, tags ...tag.Tag) {
	l.logger.V(8).Info(msg, l.kv(msg, tags)...)
}

func (l serverLogger) Fatal(msg string, tags ...tag.Tag) {
	l.logger.V(8).Info(msg, l.kv(msg, tags)...)
}
