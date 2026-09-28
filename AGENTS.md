# resume-builder

MCP server for generating one-page PDF resumes from structured data with vector-search-based content selection.

## How it works

This is a stdio-based MCP server written in Go. It exposes five tools:

- `set_embedding_config` — stores OpenAI-compatible embedding endpoint config (base URL, API key, model) on disk. Must be called before `init_resume` or `search_resume`.
- `init_resume` — accepts full structured resume data, stores on disk, embeds every bullet point + skill category + education + project into a vector store. Re-init = full overwrite of data + vector store.
- `get_resume_info` — returns cached resume data + vector store stats. No embedding config needed.
- `search_resume` — embeds job description query, searches vector store, returns ranked items grouped by category. Experiences reverse chronological, bullets ranked by relevance within each.
- `generate_resume` — generates one-page PDF. Two modes: `auto` (MCP searches vector store, selects content by relevance) or `manual` (agent provides full tailored data). Template must be specified.

Data files (`resume.json`, `vectors.json`, `chunks.json`, `embedding_config.json`) are stored next to the executable, resolved via `os.Executable()`.

## Architecture

No LLM dependency. Only needs an OpenAI-compatible embedding endpoint (e.g. LM Studio). PDF generation is in-process via `go-pdf/fpdf`.

```
Agent ──stdio──► MCP Server
                   set_embedding_config     ──► embedding_config.json
                   init_resume              ──► resume.json + vectors.json + chunks.json
                   get_resume_info          ──► resume.json + stats
                   search_resume            ──► vector store search
                   generate_resume          ──► one-page PDF + trim info
```

## Vector Store

Every bullet point gets its own embedding. This makes bullets the atomic unit of selection.

| Chunk Type | What gets embedded | Metadata |
|---|---|---|
| `experience_bullet` | Bullet text | company, role, dates, location, link, bulletIndex |
| `skill_group` | `"{category}: {values}"` | category |
| `project_bullet` | Bullet text | projectName, tech, date, link, bulletIndex |
| `education` | `"{institution} - {degree}"` | institution, degree, dates, location |

Search groups results by type. Experiences always reverse chronological. Bullets ranked by cosine similarity within each experience.

## One-Page Enforcement

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
| `fahad` | Fahad's LaTeX-style resume: Times serif, darkgray body (RGB 38,38,38), section rules, two-column headings. Letter paper, ~13mm margins. Section order: Education → Skills → Experience → Projects. |

## Build

```bash
go build -o resume-builder .
```

## Run

No env vars required.

```bash
./resume-builder
```

## MCP Config

Copy `mcp-config.json` into the agent's MCP config.

## Key files

| File | Purpose |
|---|---|
| `main.go` | Entry point, resolves data dir via `os.Executable()` |
| `internal/mcpserver/server.go` | MCP server setup, 5 tool handlers |
| `internal/resume/types.go` | ResumeData, Experience, Project, SkillGroup, Education structs |
| `internal/resume/store.go` | JSON disk-backed resume Store |
| `internal/resume/validate.go` | Guard rail quota validation |
| `internal/vectorstore/types.go` | Chunk, ScoredChunk, SearchResult types |
| `internal/vectorstore/store.go` | In-memory vector store with cosine similarity, disk persistence |
| `internal/vectorstore/embed.go` | OpenAI-compatible embedding client |
| `internal/vectorstore/config.go` | Embedding config disk persistence |
| `internal/vectorstore/index.go` | Builds chunks from resume data + embeds them |
| `internal/template/interface.go` | Renderer interface, template registry |
| `internal/template/fahad.go` | Fahad template: Times serif, darkgray, two-column layout |
| `internal/generate/generate.go` | One-page enforcement loop, PDF output |
| `internal/generate/auto.go` | AutoBuild: constructs ResumeData from search results |
| `mcp-config.json` | MCP config snippet |
