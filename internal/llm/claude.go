// Package llm drives the optional semantic and document analysis passes by
// running the `claude` CLI through spawnllm. This reuses the user's Claude
// subscription without requiring an API key to be threaded through slop-cop
// directly.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	spawnllm "github.com/yasyf/spawnllm/go"
)

// Config captures the knobs for a single claude invocation.
type Config struct {
	// Model is a literal provider model id, e.g. claude-haiku-4-5-20251001.
	Model string
	// Timeout bounds each attempt; zero picks spawnllm's 180s default.
	Timeout time.Duration
}

// RunSchema invokes claude with a JSON-schema constraint on the response and
// decodes the structured payload into out. The `system` string replaces
// claude's default system prompt — appending instead leaves the
// interactive-agent framing in place, and the model answers as a coding session
// rather than performing the task. Transient provider failures retry with
// backoff inside spawnllm; ctx bounds the whole call across attempts.
func RunSchema(ctx context.Context, cfg Config, system, user string, schema json.RawMessage, out any) error {
	// UseHostConfig keeps the user's credentials reachable; spawnllm's isolated
	// mode seeds them from account.json/credentials.json, neither of which
	// exists when claude stores its login in the macOS keychain. Dir is the
	// counterweight: claude discovers a project CLAUDE.md by walking up from
	// its working directory, so running from a scratch dir keeps the analysed
	// document's own repo instructions out of the detector's context.
	scratch, err := os.MkdirTemp("", "slop-cop-llm-")
	if err != nil {
		return fmt.Errorf("claude: scratch dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	spec := spawnllm.RunSpec{
		Prompt:        user,
		Model:         cfg.Model,
		Schema:        schema,
		UseHostConfig: true,
		Dir:           scratch,
		Timeout:       cfg.Timeout,
	}
	// UseHostConfig also hands claude the host's MCP servers, and it connects to
	// every one of them before answering — 63s against 1.9s on a workstation with
	// a handful configured, which is most of a lint pass's budget spent on servers
	// a detector never calls. An empty config plus StrictMCP starts none of them.
	spec.Providers.Claude = &spawnllm.ClaudeConfig{
		SystemPrompt: system,
		MCPConfig:    `{"mcpServers":{}}`,
		StrictMCP:    true,
	}

	resp, err := spawnllm.RunOn(ctx, spawnllm.ClaudeBackend(), spec)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	// RunOn resolves the schema payload to Result.Raw: claude's terminal
	// result field, a JSON string of the structured output.
	if err := json.Unmarshal([]byte(resp.Result.Raw), out); err != nil {
		return fmt.Errorf("claude: decoding result (%w): %s", err, truncate(resp.Result.Raw, 400))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
