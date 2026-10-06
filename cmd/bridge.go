package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/bridge"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func bridgeCommand(get func() store.Store) *cobra.Command {
	var profile string
	var idle, timeout time.Duration
	c := &cobra.Command{Use: "bridge", Short: "Run an on-demand MCP stdio bridge", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("request timeout must be positive")
		}
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		b, err := bridge.New(d, s, profile, idle)
		if err != nil {
			return err
		}
		b.RequestTimeout = timeout
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return b.Run(ctx, c.InOrStdin(), c.OutOrStdout())
	}}
	c.Flags().StringVar(&profile, "profile", "cursor", "Enabled server profile")
	c.Flags().DurationVar(&idle, "idle-timeout", 3*time.Minute, "Idle child lifetime")
	c.Flags().DurationVar(&timeout, "request-timeout", 2*time.Minute, "Maximum backend request duration")
	return c
}
