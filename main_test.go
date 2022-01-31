package main_test

import (
	"context"
	"testing"

	main "github.com/sevein/chesstempo"
)

func MustRunMain(tb testing.TB) *main.Main {
	tb.Helper()

	m := main.NewMain()
	if err := m.Run(context.Background()); err != nil {
		tb.Fatal(err)
	}

	return m
}

func MustCloseMain(tb testing.TB, m *main.Main) {
	tb.Helper()

	if err := m.Close(); err != nil {
		tb.Fatal(err)
	}
}

func TestStart(t *testing.T) {
	t.Parallel()

	m := MustRunMain(t)
	defer MustCloseMain(t, m)
}
