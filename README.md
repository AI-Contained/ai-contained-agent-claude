# ai-contained-agent-claude

A minimal, isolated Docker container for running [Claude Code](https://code.claude.com) as a non-root user with no access to your host filesystem except through explicit MCP servers.

## Security Model

- Runs as a non-root user (`65533:65533` by default, overridable at runtime)
- Based on `scratch` - contains only the claude binary and musl libs
- No shell, no package manager, no tools
- Working directory (`/ai_contained`) is empty and isolated - use MCPs to access your files
- Config/credentials are mounted via a volume, not baked into the image
- `settings.json` denies all built-in tools (Read, Write, Bash, WebSearch, etc.) - claude can only act through MCP servers you explicitly provide

## First-Time Setup (Bootstrap)

Copy the provided `template-config` as your personal config directory:

```bash
cp -r template-config claude-config
```

This creates a `claude-config/` directory containing:
- `.claude.json` - session state (will be populated on first login)
- `settings.json` - disables all built-in claude tools

> You only need to do this once. After that, `claude-config` persists your session and credentials across container runs.

## Building

```bash
docker build -t ai-contained-agent-claude:1.0.0 .
```

Pin a specific Claude version:

```bash
docker build --build-arg CLAUDE_VERSION=2.1.114 -t ai-contained-agent-claude:1.0.0 .
```

## Running

```bash
docker run -it --rm \
  --user $(id -u):$(id -g) \
  -v $(pwd)/claude-config:/config \
  ai-contained-agent-claude:1.0.0
```

Resume a previous session:

```bash
docker run -it --rm \
  --user $(id -u):$(id -g) \
  -v $(pwd)/claude-config:/config \
  ai-contained-agent-claude:1.0.0 --resume <session-id>
```

## Connecting MCP Servers

```bash
docker run -it --rm \
  --user $(id -u):$(id -g) \
  --network mcp-network \
  -v $(pwd)/claude-config:/config \
  ai-contained-agent-claude:1.0.0 \
  --strict-mcp-config \
  --mcp-config '{"mcpServers":{"my-mcp":{"type":"http","url":"http://my-mcp-server:8080/mcp"}}}'
```
