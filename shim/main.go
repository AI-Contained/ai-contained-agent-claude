package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// mirror walks srcRoot and copies anything missing from dstRoot, preserving
// permissions. Each created path is logged. Existing dst entries are left
// untouched.
func mirror(srcRoot, dstRoot string, log io.Writer) error {
	return filepath.WalkDir(srcRoot, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", srcPath, err)
		}
		rel, err := filepath.Rel(srcRoot, srcPath)
		if err != nil {
			return fmt.Errorf("rel %s: %w", srcPath, err)
		}
		if rel == "." {
			return nil
		}
		dstPath := filepath.Join(dstRoot, rel)
		if _, err := os.Stat(dstPath); err == nil {
      // The destination exists -- we're skipping it
			return nil
		}

		if d.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
		fmt.Fprintln(log, rel)
		return nil
	})
}

func copyDir(srcPath, dstPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", srcPath, err)
	}
	perm := info.Mode().Perm()
	if err := os.Mkdir(dstPath, perm); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstPath, err)
	}
	if err := os.Chmod(dstPath, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", dstPath, err)
	}
	return nil
}

func copyFile(srcPath, dstPath string) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", srcPath, err)
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", srcPath, err)
	}
	perm := info.Mode().Perm()
	out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", dstPath, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", srcPath, dstPath, err)
	}
	if err := os.Chmod(dstPath, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", dstPath, err)
	}
	return nil
}

// envOrExit returns the value of key, or exits if it's unset. The Dockerfile
// is the source of truth for these paths; absence is a misconfiguration.
func envOrExit(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		fmt.Fprintf(os.Stderr, "required environment variable %s not set\n", key)
		os.Exit(1)
	}
	return v
}

func main() {
  // fetch our env vars (or exit - before we do any heavy lifting)
	claude := envOrExit("CLAUDE_BIN")
	if err := mirror(envOrExit("TEMPLATE_DIR"), envOrExit("CLAUDE_CONFIG_DIR"), os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := syscall.Exec(claude,
		append([]string{filepath.Base(claude)}, os.Args[1:]...),
		os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
