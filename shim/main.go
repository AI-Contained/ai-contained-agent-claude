package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// mirror walks src and copies anything missing from dst, preserving
// permissions. Each created path is logged to log. Existing dst entries are
// left untouched.
func mirror(src, dst string, log io.Writer) error {
	return fmt.Errorf("mirror not implemented")
}

// TODO raise an error/exception if not set (die) --
// dockerfile should be the source of truth
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	src := envOr("TEMPLATE_DIR", "/template-config")
	dst := envOr("CLAUDE_CONFIG_DIR", "/config")
	if err := mirror(src, dst, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
  // TODO expand /home/agent into $HOME
	claude := envOr("CLAUDE_BIN", "/home/agent/.local/bin/claude")
	if err := syscall.Exec(claude,
		append([]string{filepath.Base(claude)}, os.Args[1:]...),
		os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
