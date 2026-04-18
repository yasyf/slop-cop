// Package llm drives the optional semantic and document analysis passes by
// shelling out to the `claude` CLI (`claude -p --output-format json
// --json-schema ...`). This reuses the user's Claude subscription without
// requiring an API key to be threaded through slop-cop directly.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Config captures the knobs for a single claude invocation.
type Config struct {
	// Bin is the path or name of the claude CLI; empty means "claude".
	Bin string
	// Model is the model slug passed via --model, e.g. claude-haiku-4-5-20251001.
	Model string
	// Timeout bounds a single invocation; zero means no bound.
	Timeout time.Duration
	// ExtraArgs is appended before the final user-prompt argument for callers
	// who need to pass flags like --max-budget-usd.
	ExtraArgs []string
}

// resultEnvelope is the top-level shape printed by `claude -p --output-format json`.
// Fields unused by slop-cop are deliberately omitted.
type resultEnvelope struct {
	Type       string          `json:"type"`
	Subtype    string          `json:"subtype"`
	IsError    bool            `json:"is_error"`
	Result     json.RawMessage `json:"result"`
	StructuredResult json.RawMessage `json:"structured_result"`
	Error      string          `json:"error"`
	SessionID  string          `json:"session_id"`
}

// RunSchema invokes claude with a JSON-schema constraint on the response and
// decodes the structured payload into out. The `user` prompt becomes the
// positional argument; the `system` string is appended to claude's default
// system prompt via --append-system-prompt.
func RunSchema(ctx context.Context, cfg Config, system, user string, schema json.RawMessage, out any) error {
	bin := cfg.Bin
	if bin == "" {
		bin = "claude"
	}

	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	args := []string{
		"-p",
		"--output-format", "json",
		"--json-schema", string(schema),
	}
	if system != "" {
		args = append(args, "--append-system-prompt", system)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	args = append(args, cfg.ExtraArgs...)
	args = append(args, user)

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("claude: timed out after %s", cfg.Timeout)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return fmt.Errorf("claude: %w: %s", err, truncate(detail, 400))
	}

	var env resultEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		return fmt.Errorf("claude: unrecognised output (%w): %s", err, truncate(stdout.String(), 400))
	}
	if env.IsError {
		msg := env.Error
		if msg == "" {
			msg = string(env.Result)
		}
		return fmt.Errorf("claude reported error: %s", truncate(msg, 400))
	}

	payload := env.StructuredResult
	if len(payload) == 0 {
		payload = env.Result
	}
	if len(payload) == 0 {
		return errors.New("claude: empty result payload")
	}

	// `result` arrives either as a JSON object/array (validated against the
	// schema) or as a JSON string containing the serialised object. Handle
	// both cases so the caller can rely on a structured unmarshal.
	if len(payload) > 0 && payload[0] == '"' {
		var s string
		if err := json.Unmarshal(payload, &s); err != nil {
			return fmt.Errorf("claude: result envelope (%w): %s", err, truncate(string(payload), 400))
		}
		if err := json.Unmarshal([]byte(s), out); err != nil {
			return fmt.Errorf("claude: decoding result string (%w): %s", err, truncate(s, 400))
		}
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("claude: decoding result object (%w): %s", err, truncate(string(payload), 400))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
