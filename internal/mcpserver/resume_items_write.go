package mcpserver

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func handleItemsInit(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	if args.Data == nil {
		return nil, ResumeItemsOutput{}, fmt.Errorf("data is required for init")
	}
	if strings.TrimSpace(args.Data.Name) == "" {
		return nil, ResumeItemsOutput{}, fmt.Errorf("data.name is required")
	}
	tx := newInitTx(d, *args.Data)
	if err := tx.commit(); err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	stats := tx.stats()
	chunks := tx.chunkCount()
	return jsonResult(ResumeItemsOutput{
		Operation: "init",
		Message: fmt.Sprintf("Initialized resume with %d experiences, %d bullets, %d skills, %d projects, %d education entries",
			stats.Experiences, stats.Bullets, stats.Skills, stats.Projects, stats.Education),
		Stats:        &stats,
		VectorChunks: &chunks,
	})
}

func handleItemsAdd(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	tx, err := newItemTx(d)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	entry, err := tx.add(args)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	if err := tx.commit(); err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	return txResult("add", entry, tx)
}

func handleItemsUpdate(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	tx, err := newItemTx(d)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	entry, err := tx.update(args)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	if err := tx.commit(); err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	return txResult("update", entry, tx)
}

func handleItemsDelete(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	tx, err := newItemTx(d)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	entry, err := tx.delete(args)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	if err := tx.commit(); err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	return txResult("delete", entry, tx)
}

// handleItemsBatch applies requests in order and commits them together; any
// failure discards the whole batch.
func handleItemsBatch(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	if len(args.Requests) == 0 {
		return nil, ResumeItemsOutput{}, fmt.Errorf("requests is required for batch (one or more add/update/delete requests)")
	}
	tx, err := newItemTx(d)
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	entries := make([]ItemEntry, 0, len(args.Requests))
	for i, req := range args.Requests {
		entry, err := tx.apply(req.toInput())
		if err != nil {
			return nil, ResumeItemsOutput{}, fmt.Errorf("request %d (%s %s): %w", i, req.Operation, req.Type, err)
		}
		entries = append(entries, entry)
	}
	if err := tx.commit(); err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	stats := tx.stats()
	chunks := tx.chunkCount()
	return jsonResult(ResumeItemsOutput{
		Operation:    "batch",
		Message:      fmt.Sprintf("Applied %d operations", len(entries)),
		Results:      entries,
		Stats:        &stats,
		VectorChunks: &chunks,
	})
}
