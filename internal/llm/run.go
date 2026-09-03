// Package llm drives the optional semantic and document analysis passes by
// running a provider CLI through spawnllm: codex for the detection tiers,
// claude for the rewrite prompts. This reuses the user's existing logins
// without requiring an API key to be threaded through slop-cop directly.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	spawnllm "github.com/yasyf/spawnllm/go"
)

// Config captures the knobs for a single provider invocation.
type Config struct {
	// Provider names the CLI spawnllm drives.
	Provider spawnllm.Provider
	// Model is a literal provider model id, e.g. gpt-5.6-luna:low.
	Model string
	// Timeout bounds each attempt; zero picks spawnllm's 180s default.
	Timeout time.Duration
}

// RunSchema invokes the configured provider with a JSON-schema constraint on
// the response and decodes the structured payload into out. The `system`
// string replaces the CLI's default system prompt — appending instead leaves
// the interactive-agent framing in place, and the model answers as a coding
// session rather than performing the task. Transient provider failures retry
// with backoff inside spawnllm; ctx bounds the whole call across attempts.
func RunSchema(ctx context.Context, cfg Config, system, user string, schema json.RawMessage, out any) error {
	// UseHostConfig keeps the user's credentials reachable; spawnllm's isolated
	// mode seeds them from account.json/credentials.json, neither of which
	// exists when claude stores its login in the macOS keychain. Dir is the
	// counterweight: both CLIs walk up from their working directory for a
	// CLAUDE.md or AGENTS.md, so a scratch dir keeps the analysed document's
	// own repo instructions out of the detector's context.
	scratch, err := os.MkdirTemp("", "slop-cop-llm-")
	if err != nil {
		return fmt.Errorf("llm: scratch dir: %w", err)
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

	var backend spawnllm.Backend
	switch cfg.Provider {
	case spawnllm.ProviderCodex:
		backend = spawnllm.CodexBackend()
		// The config carries only the system prompt: spawnllm already plans
		// --sandbox read-only, service_tier=fast, and features.hooks and
		// features.mcp_servers off, so a detector starts no MCP server.
		spec.Providers.Codex = &spawnllm.CodexConfig{DeveloperInstructions: system}
	case spawnllm.ProviderClaude:
		backend = spawnllm.ClaudeBackend()
		// UseHostConfig also hands claude the host's MCP servers, and it connects to
		// every one of them before answering — 63s against 1.9s on a workstation with
		// a handful configured, which is most of a lint pass's budget spent on servers
		// a detector never calls. An empty config plus StrictMCP starts none of them.
		spec.Providers.Claude = &spawnllm.ClaudeConfig{
			SystemPrompt: system,
			MCPConfig:    `{"mcpServers":{}}`,
			StrictMCP:    true,
		}
	default:
		panic(fmt.Sprintf("llm: unsupported provider %q", cfg.Provider))
	}

	resp, err := spawnllm.RunOn(ctx, backend, spec)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	// RunOn resolves the schema payload to Result.Raw: the CLI's terminal
	// result field, a JSON string of the structured output.
	if err := json.Unmarshal([]byte(resp.Result.Raw), out); err != nil {
		return fmt.Errorf("llm: decoding result (%w): %s", err, truncate(resp.Result.Raw, 400))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
