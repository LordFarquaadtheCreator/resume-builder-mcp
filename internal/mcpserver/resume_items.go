package mcpserver

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// --- resume_items tool ---

type ResumeItemsInput struct {
	Operation  string             `json:"operation" jsonschema:"required,Operation: 'init', 'add', 'get', 'update', 'delete', 'batch', or 'search'"`
	Type       string             `json:"type,omitempty" jsonschema:"Item type for add/update/delete, or optional filter for get: 'experience', 'project', 'education', 'skill', or 'bullet'. For 'bullet', parentType and parentId locate the parent and id is the bullet index."`
	ID         *int               `json:"id,omitempty" jsonschema:"Item index for get/update/delete. For 'bullet' this is the bullet index inside the parent. Use get to discover indices."`
	ParentType string             `json:"parentType,omitempty" jsonschema:"Parent item type for bullet operations: 'experience' or 'project'."`
	ParentID   *int               `json:"parentId,omitempty" jsonschema:"Index of the parent item for bullet operations."`
	Item       *ItemInput         `json:"item,omitempty" jsonschema:"Payload for add/update. Fields depend on type. experience: company, role, start, end, location, link, bullets; project: name, tech, date, link, bullets; education: institution, degree, start, end, location, link; skill: category, values; bullet: text."`
	Data       *resume.ResumeData `json:"data,omitempty" jsonschema:"Full resume payload for operation 'init': name, contact, education, skills, experiences, projects. Overwrites all existing data and rebuilds the vector store."`
	Requests   []BatchRequest     `json:"requests,omitempty" jsonschema:"Requests for operation 'batch': add/update/delete operations applied in order and committed together. If any request fails, the whole batch is discarded."`
	Query      string             `json:"query,omitempty" jsonschema:"Job description text for operation 'search'."`
	TopK       int                `json:"topK,omitempty" jsonschema:"Max items per category for 'search'. Defaults to 10."`
}

type BatchRequest struct {
	Operation  string     `json:"operation" jsonschema:"required,Operation for this request: 'add', 'update', or 'delete'"`
	Type       string     `json:"type,omitempty" jsonschema:"Item type: 'experience', 'project', 'education', 'skill', or 'bullet'."`
	ID         *int       `json:"id,omitempty" jsonschema:"Item index. For 'bullet' this is the bullet index inside the parent."`
	ParentType string     `json:"parentType,omitempty" jsonschema:"Parent item type for bullet operations: 'experience' or 'project'."`
	ParentID   *int       `json:"parentId,omitempty" jsonschema:"Index of the parent item for bullet operations."`
	Item       *ItemInput `json:"item,omitempty" jsonschema:"Payload for add/update; same fields as the single-operation 'item'."`
}

func (r BatchRequest) toInput() ResumeItemsInput {
	return ResumeItemsInput{
		Operation:  r.Operation,
		Type:       r.Type,
		ID:         r.ID,
		ParentType: r.ParentType,
		ParentID:   r.ParentID,
		Item:       r.Item,
	}
}

// ItemInput is a flat payload; only fields relevant to the target type are read.
type ItemInput struct {
	Company     string   `json:"company,omitempty" jsonschema:"Experience: company name."`
	Role        string   `json:"role,omitempty" jsonschema:"Experience: job title."`
	Start       string   `json:"start,omitempty" jsonschema:"Experience/Education: start date, e.g. 'Jan. 2024'."`
	End         string   `json:"end,omitempty" jsonschema:"Experience/Education: end date, e.g. 'Present'."`
	Location    string   `json:"location,omitempty" jsonschema:"Experience/Education: location."`
	Link        string   `json:"link,omitempty" jsonschema:"Experience/Project/Education: URL."`
	Bullets     []string `json:"bullets,omitempty" jsonschema:"Experience/Project: bullet points, plain text. On update this replaces the whole bullet list."`
	Name        string   `json:"name,omitempty" jsonschema:"Project: project name."`
	Tech        string   `json:"tech,omitempty" jsonschema:"Project: technologies used."`
	Date        string   `json:"date,omitempty" jsonschema:"Project: date, e.g. 'Mar. 2025'."`
	Institution string   `json:"institution,omitempty" jsonschema:"Education: institution name."`
	Degree      string   `json:"degree,omitempty" jsonschema:"Education: degree."`
	Category    string   `json:"category,omitempty" jsonschema:"Skill: category name, e.g. 'Languages'."`
	Values      string   `json:"values,omitempty" jsonschema:"Skill: comma-separated values, e.g. 'Go, Python, TypeScript'."`
	Text        string   `json:"text,omitempty" jsonschema:"Bullet: bullet text for add/update of type 'bullet'."`
}

// ItemEntry is one resume item plus the indices needed to address it again.
type ItemEntry struct {
	Type       string             `json:"type"`
	ID         int                `json:"id"`
	ParentType string             `json:"parentType,omitempty"`
	ParentID   *int               `json:"parentId,omitempty"`
	Experience *resume.Experience `json:"experience,omitempty"`
	Project    *resume.Project    `json:"project,omitempty"`
	Education  *resume.Education  `json:"education,omitempty"`
	Skill      *resume.SkillGroup `json:"skill,omitempty"`
	Bullet     string             `json:"bullet,omitempty"`
}

type ResumeItemsOutput struct {
	Operation    string                    `json:"operation"`
	Message      string                    `json:"message"`
	Items        []ItemEntry               `json:"items,omitempty"`   // get
	Item         *ItemEntry                `json:"item,omitempty"`    // add/update/delete
	Results      []ItemEntry               `json:"results,omitempty"` // batch
	Result       *vectorstore.SearchResult `json:"result,omitempty"`  // search
	Stats        *resume.Stats             `json:"stats,omitempty"`   // mutations
	VectorChunks *int                      `json:"vectorChunks,omitempty"`
}

func handleResumeItems(ctx context.Context, req *mcp.CallToolRequest, args ResumeItemsInput, d deps) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	log.Printf("resume_items: operation=%s type=%s id=%s parentType=%s parentId=%s requests=%d",
		args.Operation, args.Type, fmtID(args.ID), args.ParentType, fmtID(args.ParentID), len(args.Requests))

	switch args.Operation {
	case "init":
		return handleItemsInit(args, d)
	case "get":
		return handleItemsGet(args, d)
	case "add":
		return handleItemsAdd(args, d)
	case "update":
		return handleItemsUpdate(args, d)
	case "delete":
		return handleItemsDelete(args, d)
	case "batch":
		return handleItemsBatch(args, d)
	case "search":
		return handleItemsSearch(args, d)
	case "":
		return nil, ResumeItemsOutput{}, fmt.Errorf("operation is required (init, add, get, update, delete, batch, or search)")
	default:
		return nil, ResumeItemsOutput{}, fmt.Errorf("invalid operation %q (must be init, add, get, update, delete, batch, or search)", args.Operation)
	}
}

func fmtID(id *int) string {
	if id == nil {
		return "-"
	}
	return strconv.Itoa(*id)
}

// --- ItemInput conversions ---

func (in *ItemInput) toExperience() resume.Experience {
	return resume.Experience{
		Company:  in.Company,
		Role:     in.Role,
		Start:    in.Start,
		End:      in.End,
		Location: in.Location,
		Link:     in.Link,
		Bullets:  in.Bullets,
	}
}

func (in *ItemInput) toProject() resume.Project {
	return resume.Project{
		Name:    in.Name,
		Tech:    in.Tech,
		Date:    in.Date,
		Link:    in.Link,
		Bullets: in.Bullets,
	}
}

func (in *ItemInput) toEducation() resume.Education {
	return resume.Education{
		Institution: in.Institution,
		Degree:      in.Degree,
		Start:       in.Start,
		End:         in.End,
		Location:    in.Location,
		Link:        in.Link,
	}
}

func (in *ItemInput) toSkill() resume.SkillGroup {
	return resume.SkillGroup{
		Category: in.Category,
		Values:   in.Values,
	}
}

// --- ItemEntry constructors ---

func experienceEntry(index int, e resume.Experience) ItemEntry {
	item := e
	return ItemEntry{Type: resume.ItemExperience, ID: index, Experience: &item}
}

func projectEntry(index int, p resume.Project) ItemEntry {
	item := p
	return ItemEntry{Type: resume.ItemProject, ID: index, Project: &item}
}

func educationEntry(index int, e resume.Education) ItemEntry {
	item := e
	return ItemEntry{Type: resume.ItemEducation, ID: index, Education: &item}
}

func skillEntry(index int, s resume.SkillGroup) ItemEntry {
	item := s
	return ItemEntry{Type: resume.ItemSkill, ID: index, Skill: &item}
}

func bulletEntry(parentType string, parentIndex, bulletIndex int, text string) ItemEntry {
	parent := parentIndex
	return ItemEntry{Type: resume.ItemBullet, ID: bulletIndex, ParentType: parentType, ParentID: &parent, Bullet: text}
}

func allExperienceEntries(data resume.ResumeData) []ItemEntry {
	entries := make([]ItemEntry, len(data.Experiences))
	for i, e := range data.Experiences {
		entries[i] = experienceEntry(i, e)
	}
	return entries
}

func allProjectEntries(data resume.ResumeData) []ItemEntry {
	entries := make([]ItemEntry, len(data.Projects))
	for i, p := range data.Projects {
		entries[i] = projectEntry(i, p)
	}
	return entries
}

func allEducationEntries(data resume.ResumeData) []ItemEntry {
	entries := make([]ItemEntry, len(data.Education))
	for i, e := range data.Education {
		entries[i] = educationEntry(i, e)
	}
	return entries
}

func allSkillEntries(data resume.ResumeData) []ItemEntry {
	entries := make([]ItemEntry, len(data.Skills))
	for i, s := range data.Skills {
		entries[i] = skillEntry(i, s)
	}
	return entries
}

// selectEntries returns everything when id is nil, or the single entry at id.
func selectEntries[T any](id *int, all []ItemEntry, at func(int) (T, error), build func(int, T) ItemEntry) ([]ItemEntry, error) {
	if id == nil {
		return all, nil
	}
	item, err := at(*id)
	if err != nil {
		return nil, err
	}
	return []ItemEntry{build(*id, item)}, nil
}

// --- shared validation ---

func requireID(args ResumeItemsInput) (int, error) {
	if args.ID == nil {
		return 0, fmt.Errorf("id is required for %s", args.Operation)
	}
	return *args.ID, nil
}

// bulletParent requires parentType and parentId.
func bulletParent(args ResumeItemsInput) (string, int, error) {
	if args.ParentType == "" {
		return "", 0, fmt.Errorf("parentType is required for bullet operations ('experience' or 'project')")
	}
	if args.ParentID == nil {
		return "", 0, fmt.Errorf("parentId is required for bullet operations")
	}
	return args.ParentType, *args.ParentID, nil
}

func invalidTypeError(t string) error {
	return fmt.Errorf("invalid type %q (must be experience, project, education, skill, or bullet)", t)
}
