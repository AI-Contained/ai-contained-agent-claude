# ai-contained-agent-claude

A minimal, isolated Docker container for running [Claude Code](https://code.claude.com) as a non-root user with no access to your host filesystem except through explicit MCP servers.

## Security Model

- Runs as a non-root user (`65533:65533` by default, overridable at runtime)
- Based on `scratch` — contains only the claude binary, the goshim entrypoint, musl libs, and a read-only template config
- No shell, no package manager, no tools
- Working directory (`/ai_contained`) is empty and isolated — use MCPs to access your files
- Config/credentials are mounted via a volume, not baked into the image
- `settings.json` denies all built-in tools (Read, Write, Bash, WebSearch, etc.) — claude can only act through MCP servers you explicitly provide

## How it Works (goshim)

The container's entrypoint is **goshim**, a tiny static Go binary that:

1. Walks the read-only template at `/template-config` (baked into the image at build time).
2. For each entry missing in the user-mounted `/config` volume, copies it across, preserving permissions and logging `bootstrap(copy): <path>`.
3. `execve`s claude with the original args, replacing itself with the claude process.

### Why does goshim exist?

Claude requires `${CLAUDE_CONFIG_DIR}/.claude.json` to exist before it will start. **If the file is missing, claude hangs indefinitely with no error.** Earlier versions of this image required the user to manually `cp -r template-config claude-config` before the first run; goshim makes that bootstrap automatic and idempotent. On every container start it tops up `/config` with anything missing from the template — so a fresh volume gets fully populated, and existing sessions are left untouched.

The shim is purpose-built (no flags, hard-coded paths via env vars), so re-execing it provides no useful primitive to a compromised claude — it just no-ops and re-execs claude.

### Configuration

goshim is driven entirely by env vars set in the Dockerfile (it exits if any are missing):

| Variable            | Default                          | Purpose                          |
|---------------------|----------------------------------|----------------------------------|
| `TEMPLATE_DIR`      | `/template-config`               | Read-only template baked in image |
| `CLAUDE_CONFIG_DIR` | `/config`                        | User-mounted volume (writable)   |
| `CLAUDE_BIN`        | `/home/agent/.local/bin/claude`  | Claude binary to exec             |

## Building

```bash
docker build -t ai-contained-agent-claude:1.0.0 .
```

Pin a specific Claude version:

```bash
docker build --build-arg CLAUDE_VERSION=2.1.114 -t ai-contained-agent-claude:1.0.0 .
```

The build runs the goshim test suite as part of the multi-stage build — failed tests fail the docker build.

## Running

```bash
docker run -it --rm \
  --user $(id -u):$(id -g) \
  -v $(pwd)/claude-config:/config \
  ai-contained-agent-claude:1.0.0
```

The volume `claude-config` may be empty on first run — goshim populates it from the bundled template before claude starts.

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

## Developing the shim

The Go source for goshim lives under `goshim/`. Tests use [Ginkgo v2](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/).

```bash
cd goshim
go mod download      # fetch dependencies (Ginkgo, Gomega) into the module cache
go test ./...        # run the spec suite
go build ./...       # compile a host-arch binary for local inspection
```

(`go mod tidy` works too, and additionally rewrites `go.mod`/`go.sum` to match imports — use it if you've added or removed dependencies.)

Inside the docker build, the `goshim-builder` stage runs `go test ./...` before `go build`, so any test failure aborts the image build. Cross-compilation is native (no QEMU emulation): the builder pins to `$BUILDPLATFORM` and Go cross-compiles to `$TARGETOS/$TARGETARCH` via `CGO_ENABLED=0`.
