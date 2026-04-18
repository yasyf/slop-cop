package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// readInput loads text from a file path (or stdin when path is "" or "-").
func readInput(path string) (string, error) {
	if path == "" || path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(b), nil
}

// writeJSON serialises v to stdout, optionally indented for human
// inspection. Always terminates with a newline.
func writeJSON(v any, pretty bool) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}
