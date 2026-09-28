package mcpserver

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

func handleItemsGet(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	stored, err := d.ResumeStore.Load()
	if err != nil {
		return nil, ResumeItemsOutput{}, err
	}
	data := stored.Data

	var items []ItemEntry
	if args.Type == "" {
		if args.ID != nil {
			return nil, ResumeItemsOutput{}, fmt.Errorf("type is required when id is given")
		}
		// Template order: education, skills, experience, projects.
		items = append(items, allEducationEntries(data)...)
		items = append(items, allSkillEntries(data)...)
		items = append(items, allExperienceEntries(data)...)
		items = append(items, allProjectEntries(data)...)
	} else {
		if !resume.ValidItemType(args.Type) {
			return nil, ResumeItemsOutput{}, invalidTypeError(args.Type)
		}
		switch args.Type {
		case resume.ItemExperience:
			items, err = selectEntries(args.ID, allExperienceEntries(data), data.ExperienceAt, experienceEntry)
		case resume.ItemProject:
			items, err = selectEntries(args.ID, allProjectEntries(data), data.ProjectAt, projectEntry)
		case resume.ItemEducation:
			items, err = selectEntries(args.ID, allEducationEntries(data), data.EducationAt, educationEntry)
		case resume.ItemSkill:
			items, err = selectEntries(args.ID, allSkillEntries(data), data.SkillAt, skillEntry)
		case resume.ItemBullet:
			items, err = bulletEntries(args, data)
		}
		if err != nil {
			return nil, ResumeItemsOutput{}, err
		}
	}

	return jsonResult(ResumeItemsOutput{
		Operation: "get",
		Message:   fmt.Sprintf("Returned %d item(s)", len(items)),
		Items:     items,
	})
}

func bulletEntries(args ResumeItemsInput, data resume.ResumeData) ([]ItemEntry, error) {
	parentType, parentIndex, err := bulletParent(args)
	if err != nil {
		return nil, err
	}
	bullets, err := data.Bullets(parentType, parentIndex)
	if err != nil {
		return nil, err
	}
	if args.ID == nil {
		items := make([]ItemEntry, len(bullets))
		for i, b := range bullets {
			items[i] = bulletEntry(parentType, parentIndex, i, b)
		}
		return items, nil
	}
	index := *args.ID
	if index < 0 || index >= len(bullets) {
		return nil, fmt.Errorf("bullet index %d out of range (have %d)", index, len(bullets))
	}
	return []ItemEntry{bulletEntry(parentType, parentIndex, index, bullets[index])}, nil
}

func handleItemsSearch(args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	if args.Query == "" {
		return nil, ResumeItemsOutput{}, fmt.Errorf("query is required for search")
	}

	cfg, err := d.ConfigStore.Load()
	if err != nil {
		return nil, ResumeItemsOutput{}, fmt.Errorf("embedding config required: %w", err)
	}
	if !d.VectorStore.HasData() {
		return nil, ResumeItemsOutput{}, fmt.Errorf(`no vector store data — call resume_items with operation "init" first`)
	}

	queryEmb, err := vectorstore.NewEmbedClient(*cfg).Embed(args.Query)
	if err != nil {
		return nil, ResumeItemsOutput{}, fmt.Errorf("embed query: %w", err)
	}

	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}
	result := d.VectorStore.SearchGrouped(queryEmb, topK)
	return jsonResult(ResumeItemsOutput{
		Operation: "search",
		Message: fmt.Sprintf("Matched %d experiences, %d skills, %d projects, %d education entries",
			len(result.Experiences), len(result.Skills), len(result.Projects), len(result.Education)),
		Result: &result,
	})
}
