package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/absurek/flow"
)

func main() {
	cfg, err := flow.ParseConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	app, err := flow.Listen(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("listening on %s\n", app.URL())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	if err := app.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
