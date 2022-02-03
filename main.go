package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/sevein/chesstempo/game"
	"github.com/sevein/chesstempo/http"
	"github.com/sevein/chesstempo/temporal"

	"github.com/go-logr/stdr"
	"go.temporal.io/sdk/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-ctx.Done()

	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type Main struct {
	Temporal       *temporal.Client
	TemporalWorker worker.Worker
	HTTPServer     *http.Server
}

func NewMain() *Main {
	return &Main{
		Temporal:   temporal.New(),
		HTTPServer: http.NewServer(),
	}
}

const (
	namespace = "default"
	embedded  = true
	ephemeral = false
	taskQueue = "queue"
	addr      = ":9999"
)

func (m *Main) Run(ctx context.Context) error {
	stdr.SetVerbosity(7)
	logger := stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags), stdr.Options{LogCaller: stdr.All})
	logger = logger.WithName("chesstempo")

	// Start Temporal client.
	m.Temporal.Namespace = namespace
	m.Temporal.Embedded = embedded
	m.Temporal.Ephemeral = ephemeral
	if err := m.Temporal.Create(logger.WithName("temporal")); err != nil {
		return fmt.Errorf("failed to create Temporal client: %v", err)
	}

	// Start worker.
	w := worker.New(m.Temporal.Client, taskQueue, worker.Options{})
	if err := w.Start(); err != nil {
		return err
	}
	w.RegisterWorkflow(game.GameWorkflow)

	// Start HTTP server.
	m.HTTPServer.TemporalClient = m.Temporal.Client
	m.HTTPServer.Addr = addr
	if err := m.HTTPServer.Open(); err != nil {
		return fmt.Errorf("failed to create web server: %v", err)
	}
	logger.Info("HTTP server listening", "addr", m.HTTPServer.Addr)

	go func() { http.ListenAndServeDebug() }()

	return nil
}

func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}

	if m.TemporalWorker != nil {
		m.TemporalWorker.Stop()
	}

	if m.Temporal != nil {
		if err := m.Temporal.Close(); err != nil {
			return err
		}
	}

	return nil
}
