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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	src := envOr("SRC_TEMPLATE_CONFIG", "/opt/template-config")
	dst := envOr("DST_TEMPLATE_CONFIG", "/config")
	if err := mirror(src, dst, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	claude := envOr("CLAUDE_PATH", "/home/agent/.local/bin/claude")
	if err := syscall.Exec(claude,
		append([]string{filepath.Base(claude)}, os.Args[1:]...),
		os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
