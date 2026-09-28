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

type InitResumeInput struct {
	Data resume.ResumeData `json:"data" jsonschema:"required,Full structured resume data"`
}

type GetResumeInfoInput struct{}

type SearchResumeInput struct {
	Query string `json:"query" jsonschema:"required,Job description or query text to search against"`
	TopK  int    `json:"topK,omitempty" jsonschema:"Max items per category. Defaults to 10."`
}

type GenerateResumeInput struct {
	Mode      string             `json:"mode" jsonschema:"required,Generation mode: 'auto' (MCP selects content) or 'manual' (agent provides full data)"`
	Query     string             `json:"query,omitempty" jsonschema:"Job description for auto mode. Required if mode is 'auto'."`
	Data      *resume.ResumeData `json:"data,omitempty" jsonschema:"Full resume data for manual mode. Required if mode is 'manual'."`
	Template  string             `json:"template" jsonschema:"required,Template name (e.g. 'fahad')"`
	OutputDir string             `json:"outputDir,omitempty" jsonschema:"Output directory. Defaults to /tmp."`
}

// --- Tool outputs ---

type SetEmbeddingConfigOutput struct {
	Message string `json:"message"`
}

type InitResumeOutput struct {
	Message string       `json:"message"`
	Stats   resume.Stats `json:"stats"`
}

type GetResumeInfoOutput struct {
	Resume        *resume.StoredResume `json:"resume"`
	Stats         resume.Stats         `json:"stats"`
	VectorChunks  int                  `json:"vectorChunks"`
	HasEmbedding  bool                 `json:"hasEmbeddingConfig"`
	InitializedAt string               `json:"initializedAt,omitempty"`
}

type SearchResumeOutput struct {
	Result vectorstore.SearchResult `json:"result"`
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
		Description: "Set the embedding provider configuration. MUST be called before init_resume or search_resume. Stores an OpenAI-compatible embedding endpoint (base URL, API key, model name). Persists on disk across restarts.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SetEmbeddingConfigInput) (*mcp.CallToolResult, SetEmbeddingConfigOutput, error) {
		return handleSetEmbeddingConfig(ctx, req, args, d)
	})

	// 2. init_resume
	mcp.AddTool(server, &mcp.Tool{
		Name:        "init_resume",
		Description: "Initialize or re-initialize stored resume data. Accepts full structured resume (name, contact, education, skills, experiences, projects). Embeds every bullet point and skill category into a vector store for relevance-based search. Requires set_embedding_config to be called first. Re-init overwrites all existing data and rebuilds the vector store.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args InitResumeInput) (*mcp.CallToolResult, InitResumeOutput, error) {
		return handleInitResume(ctx, req, args, d)
	})

	// 3. get_resume_info
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_resume_info",
		Description: "Get cached resume data and vector store stats. No embedding config needed. Returns the full stored resume, content counts, and whether embedding config is set.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetResumeInfoInput) (*mcp.CallToolResult, GetResumeInfoOutput, error) {
		return handleGetResumeInfo(ctx, req, d)
	})

	// 4. search_resume
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_resume",
		Description: "Search resume vector store by job description. Returns most relevant items grouped by category: experiences (reverse chronological, bullets ranked by relevance), skills, projects, education. Use this in manual mode to let the agent select and tailor content before calling generate_resume.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchResumeInput) (*mcp.CallToolResult, SearchResumeOutput, error) {
		return handleSearchResume(ctx, req, args, d)
	})

	// 5. generate_resume
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_resume",
		Description: "Generate a one-page PDF resume. Two modes: 'auto' (MCP searches vector store and selects content based on job description query) or 'manual' (agent provides full tailored resume data). Template must be specified (e.g. 'fahad'). Output saved to outputDir (default /tmp) as <Name>Resume.pdf. One-page enforced via measurement loop: trims oldest/lowest-relevance bullets, then experiences, then projects, then font scaling as last resort. Returns what was dropped.",
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
		Message: "Embedding config saved. You can now call init_resume.",
	})
}

func handleInitResume(ctx context.Context, req *mcp.CallToolRequest, args InitResumeInput, d deps) (*mcp.CallToolResult, InitResumeOutput, error) {
	log.Printf("init_resume: name=%s experiences=%d skills=%d projects=%d education=%d",
		args.Data.Name, len(args.Data.Experiences), len(args.Data.Skills), len(args.Data.Projects), len(args.Data.Education))

	// Require embedding config
	embCfg, err := d.ConfigStore.Load()
	if err != nil {
		log.Printf("init_resume: ERROR no embedding config: %v", err)
		return nil, InitResumeOutput{}, fmt.Errorf("embedding config required: %w", err)
	}

	log.Printf("init_resume: skipping quota validation — no max limits on init")

	// Save resume data
	if err := d.ResumeStore.Save(args.Data); err != nil {
		log.Printf("init_resume: ERROR saving resume: %v", err)
		return nil, InitResumeOutput{}, err
	}
	log.Printf("init_resume: resume data saved")

	// Build vector store
	embedClient := vectorstore.NewEmbedClient(*embCfg)
	log.Printf("init_resume: embedding %d chunks via %s", countChunks(args.Data), embCfg.Model)
	chunks, err := vectorstore.IndexResume(args.Data, embedClient)
	if err != nil {
		log.Printf("init_resume: ERROR indexing resume: %v", err)
		return nil, InitResumeOutput{}, fmt.Errorf("index resume: %w", err)
	}
	log.Printf("init_resume: embedded %d chunks", len(chunks))

	// Reset and rebuild vector store
	d.VectorStore.Reset()
	for _, c := range chunks {
		d.VectorStore.AddChunk(c)
	}
	if err := d.VectorStore.Save(); err != nil {
		log.Printf("init_resume: ERROR saving vector store: %v", err)
		return nil, InitResumeOutput{}, fmt.Errorf("save vector store: %w", err)
	}

	stats := resume.ComputeStats(args.Data)
	log.Printf("init_resume: success — %d experiences, %d bullets, %d skills, %d projects, %d education",
		stats.Experiences, stats.Bullets, stats.Skills, stats.Projects, stats.Education)
	return jsonResult(InitResumeOutput{
		Message: fmt.Sprintf("Resume initialized with %d experiences, %d bullets, %d skills, %d projects, %d education entries", stats.Experiences, stats.Bullets, stats.Skills, stats.Projects, stats.Education),
		Stats:   stats,
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

func handleSearchResume(ctx context.Context, req *mcp.CallToolRequest, args SearchResumeInput, d deps) (*mcp.CallToolResult, SearchResumeOutput, error) {
	log.Printf("search_resume: query=%q topK=%d", args.Query, args.TopK)
	if args.Query == "" {
		log.Printf("search_resume: ERROR empty query")
		return nil, SearchResumeOutput{}, fmt.Errorf("query is required")
	}

	embCfg, err := d.ConfigStore.Load()
	if err != nil {
		log.Printf("search_resume: ERROR no embedding config: %v", err)
		return nil, SearchResumeOutput{}, fmt.Errorf("embedding config required: %w", err)
	}

	if !d.VectorStore.HasData() {
		log.Printf("search_resume: ERROR no vector store data")
		return nil, SearchResumeOutput{}, fmt.Errorf("no vector store data — call init_resume first")
	}

	embedClient := vectorstore.NewEmbedClient(*embCfg)
	log.Printf("search_resume: embedding query via %s", embCfg.Model)
	queryEmb, err := embedClient.Embed(args.Query)
	if err != nil {
		log.Printf("search_resume: ERROR embedding query: %v", err)
		return nil, SearchResumeOutput{}, fmt.Errorf("embed query: %w", err)
	}

	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}

	result := d.VectorStore.SearchGrouped(queryEmb, topK)
	log.Printf("search_resume: success — %d experiences, %d skills, %d projects, %d education matched",
		len(result.Experiences), len(result.Skills), len(result.Projects), len(result.Education))
	return jsonResult(SearchResumeOutput{Result: result})
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
			return nil, GenerateResumeOutput{}, fmt.Errorf("no vector store data — call init_resume first")
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

func countChunks(data resume.ResumeData) int {
	n := 0
	for _, e := range data.Experiences {
		n += len(e.Bullets)
	}
	n += len(data.Skills)
	for _, p := range data.Projects {
		n += len(p.Bullets)
	}
	n += len(data.Education)
	return n
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
