# CRUD Tools for Resume Items - Implementation Plan

## Goal
Enable incremental updates to stored resume (add/edit/delete individual items), absorb `search_resume` into unified query tool, replace `init_resume` health check with dedicated `health` tool.

## New Tools

### 1. `health` - Server status check
**Input:** none

**Output:**
```json
{
  "hasResume": bool,
  "hasEmbeddingConfig": bool,
  "vectorChunkCount": int,
  "resumeStats": {
    "experiences": int,
    "bullets": int,
    "skills": int,
    "projects": int,
    "education": int
  },
  "initializedAt": string (ISO timestamp)
}
```

**Purpose:** Tells agent if resume exists, embedding config set, data size. Replaces need to call `get_resume_info` just to check state.

### 2. `resume_items` - Unified CRUD + search
**Input structure (conditional on operation):**

```json
{
  "operation": "add" | "get" | "update" | "delete" | "search",
  // Conditional fields based on operation...
}
```

#### Operation: `add`
**Additional fields:**
- `type`: "experience" | "project" | "education" | "skill" | "bullet"
- `item`: full struct for that type
- For bullets only:
  - `parentType`: "experience" | "project"
  - `parentId`: int (index of parent in array)

**Behavior:** Updates stored resume + adds to vector store + re-embeds new item

#### Operation: `get`
**Additional fields:**
- `type`: optional filter by type
- `id`: optional specific item ID (index for items, parent+index for bullets)

**Behavior:** Returns matching items from stored resume

#### Operation: `update`
**Additional fields:**
- `type`: item type
- `id`: item identifier (index or computed key)
- `item`: partial/complete replacement

**Behavior:** Updates stored resume + vector store (re-embeds if text changed)

#### Operation: `delete`
**Additional fields:**
- `type`: item type
- `id`: item identifier

**Behavior:** Removes from stored resume + vector store

#### Operation: `search`
**Additional fields:**
- `query`: job description text
- `topK`: max per category (default 10)

**Behavior:** Returns SearchResult grouped by type (absorbs `search_resume`)

### 3. `init_resume` - First-time setup only
**Update description:** "For first-time setup only. Use resume_items for incremental updates. Call health to check current state."

**Behavior:** Unchanged (bulk overwrite)

## Remove
- `search_resume` - fully absorbed into `resume_items` operation="search"

## Implementation Changes

### 1. internal/resume/types.go
- Add ID generation helpers (index-based is fine)
- Add bullet parent reference helpers
- Add constants for item types

### 2. internal/resume/store.go
Add CRUD methods:

**Experience:**
- `AddExperience(data Experience) error`
- `GetExperience(index int) (Experience, error)`
- `UpdateExperience(index int, data Experience) error`
- `DeleteExperience(index int) error`

**Project:**
- `AddProject(data Project) error`
- `GetProject(index int) (Project, error)`
- `UpdateProject(index int, data Project) error`
- `DeleteProject(index int) error`

**Education:**
- `AddEducation(data Education) error`
- `GetEducation(index int) (Education, error)`
- `UpdateEducation(index int, data Education) error`
- `DeleteEducation(index int) error`

**Skill:**
- `AddSkill(data SkillGroup) error`
- `GetSkill(index int) (SkillGroup, error)`
- `UpdateSkill(index int, data SkillGroup) error`
- `DeleteSkill(index int) error`

**Bullet (with parent refs):**
- `AddBullet(parentType string, parentIndex int, text string) error`
- `GetBullet(parentType string, parentIndex int, bulletIndex int) (string, error)`
- `UpdateBullet(parentType string, parentIndex int, bulletIndex int, text string) error`
- `DeleteBullet(parentType string, parentIndex int, bulletIndex int) error`

All methods modify stored resume in-place.

### 3. internal/vectorstore/store.go
Add sync methods:
- `AddChunk(chunk Chunk) error`
- `UpdateChunk(id string, chunk Chunk) error`
- `DeleteChunk(id string) error`
- Re-embed on text changes
- Handle parent child relationships (bullets)

### 4. internal/mcpserver/server.go
- Add `health` tool handler
- Add `resume_items` tool handler with operation dispatch
- Remove `search_resume` tool entirely
- Update `init_resume` description
- Keep `get_resume_info` (still useful for full resume dump)

## Tests

### internal/resume/store_test.go
- Test AddExperience, GetExperience, UpdateExperience, DeleteExperience
- Test AddProject, GetProject, UpdateProject, DeleteProject
- Test AddEducation, GetEducation, UpdateEducation, DeleteEducation
- Test AddSkill, GetSkill, UpdateSkill, DeleteSkill
- Test AddBullet, GetBullet, UpdateBullet, DeleteBullet with parent refs
- Test bullet operations with invalid parent IDs
- Test ID-based lookups
- Test CRUD on empty resume
- Test CRUD with invalid indices

### internal/vectorstore/store_test.go
- Test AddChunk, UpdateChunk, DeleteChunk
- Test vector store sync with resume CRUD operations
- Test re-embedding on text changes
- Test vector store cleanup on delete
- Test chunk ID generation and lookup

### internal/mcpserver/server_test.go
- Test health tool output structure
- Test health when no resume exists
- Test health when resume exists
- Test resume_items operation="add" for all types
- Test resume_items operation="get" with filters
- Test resume_items operation="get" with specific ID
- Test resume_items operation="update" with partial data
- Test resume_items operation="delete"
- Test resume_items operation="search" (was search_resume)
- Test search still works after CRUD operations
- Test add without embedding config fails
- Test invalid operation fails
- Test missing required fields per operation
- Test add bullet with invalid parent fails
- Test delete bullet from non-existent parent fails

### E2E tests
- Test full workflow: health → init_resume → add experience → search → update bullet → delete project → generate resume
- Test search returns newly added items
- Test search excludes deleted items
- Test update changes search rankings
- Test vector store stays in sync after multiple CRUD operations

## Backward Compatibility
- `get_resume_info` unchanged
- `set_embedding_config` unchanged
- `generate_resume` unchanged
- `init_resume` unchanged (just description updated)
- Agent migration: use `health` to check state, `resume_items` for CRUD

## Edge Cases
- Bullet operations with invalid parent references
- Delete last bullet vs delete entire experience
- Update bullet vs update parent experience
- Vector store desync (add to resume but fail to embed)
- Empty resume after all deletions
- Duplicate detection (optional)
- Concurrent operations (not applicable, single-process MCP)
- Embedding failure during add/update should rollback resume change
