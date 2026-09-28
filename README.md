# resume-builder

MCP server for generating one-page PDF resumes from structured data with vector-search-based content selection.

## How it works

Stdio-based MCP server written in Go. Exposes five tools:

- `set_embedding_config` — stores OpenAI-compatible embedding endpoint config on disk. Must be called before `init_resume` or `search_resume`.
- `init_resume` — stores structured resume data, embeds every bullet point + skill category into a vector store. Re-init = full overwrite.
- `get_resume_info` — returns cached resume data + vector store stats.
- `search_resume` — searches vector store by job description. Returns ranked items grouped by category. Experiences reverse chronological, bullets ranked by relevance.
- `generate_resume` — generates one-page PDF. Two modes: `auto` (MCP selects content from vector store) or `manual` (agent provides tailored data).

No LLM dependency. Only needs an embedding endpoint (e.g. LM Studio).

## One-page enforcement

1. **Guard rail quotas** — max 6 experiences, 5 bullets/exp, 4 projects, 2 bullets/proj, 5 skill groups, 3 education entries.
2. **Measurement loop** — render to in-memory PDF, check if content fits page height. If overflow:
   - Trim last bullet from oldest experience
   - Drop oldest experience entirely
   - Trim last bullet from last project
   - Drop last project
   - Font scaling (floor at 11pt)
3. Returns `trimmed` info listing what was dropped.

## Templates

| Template | Description |
|---|---|
| `fahad` | Fahad's LaTeX-style resume: Times serif, darkgray body, section rules, two-column headings. Letter paper, ~13mm margins. |

## Build

```bash
go build -o resume-builder .
```

## Run

No env vars required. Data files (`resume.json`, `vectors.json`, `chunks.json`, `embedding_config.json`) stored next to executable.

```bash
./resume-builder
```

## MCP Config

Copy `mcp-config.json` into the agent's MCP config.

## Agent Workflow

```
1. set_embedding_config(baseUrl, model)    → one-time setup
2. init_resume(full resume data)           → stores + builds vector store
3. Agent asks user: auto or manual?
   auto:   generate_resume(mode="auto", query="job desc", template="fahad")
   manual: search_resume(query) → agent tailors → generate_resume(mode="manual", data=..., template="fahad")
```

Re-init (step 2) updates stored data + rebuilds vector store.
