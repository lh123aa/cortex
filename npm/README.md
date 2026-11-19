# cortex-mcp-local

Local-first memory MCP server for Claude Code, Cursor and any MCP client.
One 33 MB Go binary, no Docker, no database, no cloud. Your notes never leave the machine.

- **~50 ms** end-to-end ranked retrieval (measured on 557 docs; [benchmark](https://dev.to/lh123aa/same-50-ms-as-ripgrep-but-it-returns-answers-benchmarking-a-local-memory-index-on-557-docs-3nj0))
- Returns **ranked 512-token chunks**, not file paths
- Indexes markdown, text, code, PDF, DOCX, HTML
- Free tier: 5,000 documents / 1 GiB

## Run

```bash
npx -y cortex-mcp-local mcp
```

The first run downloads the right release binary for your platform into `~/.cortex/bin/`, verifies its SHA256, and starts the MCP server over stdio. Later runs start instantly.

## Claude Code

```bash
claude mcp add cortex -- npx -y cortex-mcp-local mcp
```

## Claude Desktop / Cursor

`claude_desktop_config.json` (or `.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "cortex": {
      "command": "npx",
      "args": ["-y", "cortex-mcp-local", "mcp"]
    }
  }
}
```

## Index something first

The MCP server reads the index it is given; build one with the binary directly:

```bash
npx -y cortex-mcp-local install ~/notes    # create config + index a folder
npx -y cortex-mcp-local status
```

## Licence / tiers

Free works out of the box. Pro raises the limits to 500,000 documents / 100 GiB and unlocks the console,
reranking and auto-index. A 14-day Pro trial needs only an email: <https://cortex.qataroasis.cc/trial>

## Alternatives

Prefer a plain binary? `curl -fsSL https://cortex.qataroasis.cc/install.sh | sh` (macOS/Linux),
or `irm https://cortex.qataroasis.cc/install.ps1 | iex` (Windows).

Source: <https://github.com/lh123aa/cortex> · Apache-2.0
