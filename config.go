package flow

import (
	"flag"
	"fmt"
	"io"
	"net"
)

func envPort(getenv func(string) string) string {
	if getenv == nil {
		return ""
	}

	return getenv("PORT")
}

// ParseConfig reads process arguments and the environment.
// The listen address is 127.0.0.1 and port 8080. -port wins over PORT.
func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	fs := flag.NewFlagSet("flow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.String("port", "", "port to listen on")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	listenPort := "8080"
	if *port != "" {
		listenPort = *port
	} else if p := envPort(getenv); p != "" {
		listenPort = p
	}

	addr := net.JoinHostPort("127.0.0.1", listenPort)
	resolved, err := requireLoopback(addr)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:   resolved,
		DBPath: "flow.db",
	}, nil
}
