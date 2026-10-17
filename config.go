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
// The listen address defaults to 127.0.0.1:8080. -port wins over PORT, and the host
// stays on loopback. -addr is refused unless it is already a loopback address.
func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	fs := flag.NewFlagSet("flow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.String("port", "", "port to listen on")
	addrFlag := fs.String("addr", "", "loopback address to listen on")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	addr := *addrFlag
	if addr == "" {
		listenPort := "8080"
		if *port != "" {
			listenPort = *port
		} else if p := envPort(getenv); p != "" {
			listenPort = p
		}

		addr = net.JoinHostPort("127.0.0.1", listenPort)
	}

	resolved, err := requireLoopback(addr)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:   resolved,
		DBPath: "flow.db",
	}, nil
}
