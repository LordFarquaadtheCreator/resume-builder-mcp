package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/generate"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/template"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

type deps struct {
	ResumeStore *resume.Store
	VectorStore *vectorstore.Store
	ConfigStore *vectorstore.ConfigStore
}

// --- Tool inputs ---

type SetEmbeddingConfigInput struct {
	BaseURL string `json:"baseUrl" jsonschema:"required,OpenAI-compatible embedding endpoint URL (e.g. http://localhost:1234)"`
	APIKey  string `json:"apiKey,omitempty" jsonschema:"API key for the embedding endpoint. Empty for local providers like LM Studio."`
	Model   string `json:"model" jsonschema:"required,Embedding model name (e.g. text-embedding-embeddinggemma-300m-qat)"`
}

type GetResumeInfoInput struct{}

type GenerateResumeInput struct {
	Mode      string             `json:"mode" jsonschema:"required,Generation mode: 'auto' (MCP selects content) or 'manual' (agent provides full data)"`
	Query     string             `json:"query,omitempty" jsonschema:"Job description for auto mode. Required if mode is 'auto'."`
	Data      *resume.ResumeData `json:"data,omitempty" jsonschema:"Full resume data for manual mode. Required if mode is 'manual'."`
	Template  string             `json:"template" jsonschema:"required,Template name ('fahad' or 'bennett')"`
	OutputDir string             `json:"outputDir,omitempty" jsonschema:"Output directory. Defaults to /tmp."`
}

// --- Tool outputs ---

type SetEmbeddingConfigOutput struct {
	Message string `json:"message"`
}

type GetResumeInfoOutput struct {
	Resume        *resume.StoredResume `json:"resume"`
	Stats         resume.Stats         `json:"stats"`
	VectorChunks  int                  `json:"vectorChunks"`
	HasEmbedding  bool                 `json:"hasEmbeddingConfig"`
	InitializedAt string               `json:"initializedAt,omitempty"`
}

type GenerateResumeOutput struct {
	Message    string            `json:"message"`
	OutputPath string            `json:"outputPath"`
	Filename   string            `json:"filename"`
	Trimmed    generate.TrimInfo `json:"trimmed"`
}

// Run starts the stdio MCP server.
func Run(dataDir string) error {
	d := deps{
		ResumeStore: resume.NewStore(dataDir),
		VectorStore: vectorstore.NewStore(dataDir),
		ConfigStore: vectorstore.NewConfigStore(dataDir),
	}

	// Load existing vector store data
	if err := d.VectorStore.Load(); err != nil {
		return fmt.Errorf("load vector store: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "resume-builder", Version: "1.0.0"}, nil)

	// 1. set_embedding_config
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_embedding_config",
		Description: "Set the embedding provider configuration. MUST be called before resume_items init/add/update (which embed new content) and resume_items search. Stores an OpenAI-compatible embedding endpoint (base URL, API key, model name). Persists on disk across restarts.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SetEmbeddingConfigInput) (*mcp.CallToolResult, SetEmbeddingConfigOutput, error) {
		return handleSetEmbeddingConfig(ctx, req, args, d)
	})

	// 2. health
	mcp.AddTool(server, &mcp.Tool{
		Name:        "health",
		Description: "Check server state: whether resume data exists, whether embedding config is set, vector store chunk count, resume item stats, and when the resume was initialized. No config needed. Use this first to decide between resume_items operation 'init' (first-time setup) and incremental operations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HealthInput) (*mcp.CallToolResult, HealthOutput, error) {
		return handleHealth(d)
	})

	// 3. get_resume_info
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_resume_info",
		Description: "Get the full stored resume data and vector store stats. No embedding config needed. Use resume_items type='get' with filters to read single items instead of the full dump.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetResumeInfoInput) (*mcp.CallToolResult, GetResumeInfoOutput, error) {
		return handleGetResumeInfo(ctx, req, d)
	})

	// 4. resume_items
	mcp.AddTool(server, &mcp.Tool{
		Name:        "resume_items",
		Description: "Setup, CRUD, batch edits, and search for resume items. operation 'init' (data) overwrites the whole resume and rebuilds the vector store — first-time setup; 'add' (type + item) appends one item; 'get' lists items (optional type/id filter) so you can discover indices; 'update' (type + id + item) patches non-empty fields (bullets replace the whole list when provided) and re-embeds changed text; 'delete' (type + id) removes an item; 'batch' (requests) applies multiple add/update/delete operations in order and saves once, all-or-nothing; 'search' (query) returns ranked items grouped by category (experiences reverse chronological, bullets ranked by relevance). Types: experience, project, education, skill, bullet. For bullet operations pass parentType ('experience' or 'project'), parentId, and the bullet index as id. Embedding config is required for init, add, update with changed text, and search.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ResumeItemsInput) (*mcp.CallToolResult, ResumeItemsOutput, error) {
		return handleResumeItems(ctx, req, args, d)
	})

	// 5. generate_resume
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_resume",
		Description: "Generate a one-page PDF resume. Two modes: 'auto' (MCP searches vector store and selects content based on job description query) or 'manual' (agent provides full tailored resume data). Template must be specified ('fahad' for the serif classic, 'bennett' for the modern sans-serif layout). Output saved to outputDir (default /tmp) as <Name>Resume.pdf. One-page enforced via measurement loop: trims oldest/lowest-relevance bullets, then experiences, then projects, then font scaling as last resort. Returns what was dropped.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GenerateResumeInput) (*mcp.CallToolResult, GenerateResumeOutput, error) {
		return handleGenerateResume(ctx, req, args, d)
	})

	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// --- Handlers ---

func handleSetEmbeddingConfig(ctx context.Context, req *mcp.CallToolRequest, args SetEmbeddingConfigInput, d deps) (*mcp.CallToolResult, SetEmbeddingConfigOutput, error) {
	log.Printf("set_embedding_config: baseUrl=%s model=%s", args.BaseURL, args.Model)
	cfg := vectorstore.EmbeddingConfig{
		BaseURL: args.BaseURL,
		APIKey:  args.APIKey,
		Model:   args.Model,
	}
	if err := d.ConfigStore.Save(cfg); err != nil {
		log.Printf("set_embedding_config: ERROR saving config: %v", err)
		return nil, SetEmbeddingConfigOutput{}, err
	}
	log.Printf("set_embedding_config: config saved successfully")
	return jsonResult(SetEmbeddingConfigOutput{
		Message: "Embedding config saved. You can now call resume_items.",
	})
}

func handleGetResumeInfo(ctx context.Context, req *mcp.CallToolRequest, d deps) (*mcp.CallToolResult, GetResumeInfoOutput, error) {
	log.Printf("get_resume_info: fetching cached data")
	stored, err := d.ResumeStore.Load()
	if err != nil {
		log.Printf("get_resume_info: ERROR loading resume: %v", err)
		return nil, GetResumeInfoOutput{}, err
	}

	stats := resume.ComputeStats(stored.Data)
	hasEmb := d.ConfigStore.Exists()

	out := GetResumeInfoOutput{
		Resume:       stored,
		Stats:        stats,
		VectorChunks: d.VectorStore.ChunkCount(),
		HasEmbedding: hasEmb,
	}
	if !stored.InitializedAt.IsZero() {
		out.InitializedAt = stored.InitializedAt.Format("2006-01-02T15:04:05Z")
	}

	log.Printf("get_resume_info: name=%s chunks=%d hasEmbedding=%v", stored.Data.Name, out.VectorChunks, hasEmb)
	return jsonResult(out)
}

func handleGenerateResume(ctx context.Context, req *mcp.CallToolRequest, args GenerateResumeInput, d deps) (*mcp.CallToolResult, GenerateResumeOutput, error) {
	log.Printf("generate_resume: mode=%s template=%s outputDir=%s", args.Mode, args.Template, args.OutputDir)
	if args.Template == "" {
		log.Printf("generate_resume: ERROR missing template")
		return nil, GenerateResumeOutput{}, fmt.Errorf("template is required (available: %v)", template.AvailableTemplates())
	}

	var data resume.ResumeData

	switch args.Mode {
	case "auto":
		log.Printf("generate_resume: auto mode, query=%q", args.Query)
		if args.Query == "" {
			log.Printf("generate_resume: ERROR auto mode requires query")
			return nil, GenerateResumeOutput{}, fmt.Errorf("query is required for auto mode")
		}

		// Load stored resume
		stored, err := d.ResumeStore.Load()
		if err != nil {
			log.Printf("generate_resume: ERROR loading resume: %v", err)
			return nil, GenerateResumeOutput{}, err
		}

		// Search vector store
		embCfg, err := d.ConfigStore.Load()
		if err != nil {
			log.Printf("generate_resume: ERROR no embedding config: %v", err)
			return nil, GenerateResumeOutput{}, fmt.Errorf("embedding config required for auto mode: %w", err)
		}

		if !d.VectorStore.HasData() {
			log.Printf("generate_resume: ERROR no vector store data")
			return nil, GenerateResumeOutput{}, fmt.Errorf(`no vector store data — call resume_items with operation "init" first`)
		}

		embedClient := vectorstore.NewEmbedClient(*embCfg)
		queryEmb, err := embedClient.Embed(args.Query)
		if err != nil {
			log.Printf("generate_resume: ERROR embedding query: %v", err)
			return nil, GenerateResumeOutput{}, fmt.Errorf("embed query: %w", err)
		}

		result := d.VectorStore.SearchGrouped(queryEmb, 10)
		data = generate.AutoBuild(stored.Data, result)
		log.Printf("generate_resume: auto built — %d experiences, %d projects", len(data.Experiences), len(data.Projects))

	case "manual":
		if args.Data == nil {
			log.Printf("generate_resume: ERROR manual mode requires data")
			return nil, GenerateResumeOutput{}, fmt.Errorf("data is required for manual mode")
		}
		data = *args.Data
		log.Printf("generate_resume: manual mode, name=%s experiences=%d", data.Name, len(data.Experiences))

	default:
		log.Printf("generate_resume: ERROR invalid mode %q", args.Mode)
		return nil, GenerateResumeOutput{}, fmt.Errorf("mode must be 'auto' or 'manual'")
	}

	out, err := generate.Run(data, args.Template, args.OutputDir)
	if err != nil {
		log.Printf("generate_resume: ERROR generating PDF: %v", err)
		return nil, GenerateResumeOutput{}, err
	}
	log.Printf("generate_resume: success — output=%s fitsOnePage=%v fontScale=%.2f droppedBullets=%d droppedExp=%d droppedProj=%d",
		out.OutputPath, out.Trimmed.FitsOnePage, out.Trimmed.FontScale,
		len(out.Trimmed.DroppedBullets), len(out.Trimmed.DroppedExperiences), len(out.Trimmed.DroppedProjects))
	return jsonResult(GenerateResumeOutput{
		Message:    out.Message,
		OutputPath: out.OutputPath,
		Filename:   out.Filename,
		Trimmed:    out.Trimmed,
	})
}

// jsonResult marshals the structured output as pretty JSON in the text content.
func jsonResult[T any](out T) (*mcp.CallToolResult, T, error) {
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, out, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, out, nil
}
