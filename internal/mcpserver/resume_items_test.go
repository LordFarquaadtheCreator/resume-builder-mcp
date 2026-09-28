package mcpserver

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// --- test helpers ---

func crudTestDepsAt(t *testing.T) (deps, string) {
	t.Helper()
	dir := t.TempDir()
	d := deps{
		ResumeStore: resume.NewStore(dir),
		VectorStore: vectorstore.NewStore(dir),
		ConfigStore: vectorstore.NewConfigStore(dir),
	}
	return d, dir
}

func crudTestDeps(t *testing.T) deps {
	t.Helper()
	d, _ := crudTestDepsAt(t)
	return d
}

type mockEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// mockEmbedServer implements POST /v1/embeddings with deterministic vectors.
func mockEmbedServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var resp mockEmbedResponse
		resp.Data = append(resp.Data, struct {
			Embedding []float64 `json:"embedding"`
		}{Embedding: fakeEmbedding(req.Input)})
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("mock embed encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeEmbedding hashes words into a fixed-size vector so shared vocabulary
// produces higher cosine similarity.
func fakeEmbedding(text string) []float64 {
	const dims = 64
	vec := make([]float64, dims)
	for _, raw := range strings.Fields(strings.ToLower(text)) {
		word := strings.Trim(raw, ".,;:!?()[]\"'")
		if word == "" {
			continue
		}
		h := fnv.New32a()
		h.Write([]byte(word))
		vec[int(h.Sum32()%dims)]++
	}
	return vec
}

func setMockConfig(t *testing.T, d deps, srv *httptest.Server) {
	t.Helper()
	if err := d.ConfigStore.Save(vectorstore.EmbeddingConfig{BaseURL: srv.URL, Model: "mock"}); err != nil {
		t.Fatalf("save embedding config: %v", err)
	}
}

func removeEmbeddingConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, "embedding_config.json")); err != nil {
		t.Fatalf("remove embedding config: %v", err)
	}
}

func initResumeForTest(t *testing.T, d deps, data resume.ResumeData) ResumeItemsOutput {
	t.Helper()
	_, out, err := handleResumeItems(context.Background(), &mcp.CallToolRequest{}, ResumeItemsInput{Operation: "init", Data: &data}, d)
	if err != nil {
		t.Fatalf("resume_items init: %v", err)
	}
	return out
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

func initWithMockEmbedding(t *testing.T) deps {
	t.Helper()
	d := crudTestDeps(t)
	setMockConfig(t, d, mockEmbedServer(t))
	initResumeForTest(t, d, sampleResumeData())
	return d
}

func callItems(t *testing.T, d deps, in ResumeItemsInput) ResumeItemsOutput {
	t.Helper()
	_, out, err := handleResumeItems(context.Background(), &mcp.CallToolRequest{}, in, d)
	if err != nil {
		t.Fatalf("resume_items(%s, type=%s): %v", in.Operation, in.Type, err)
	}
	return out
}

func callItemsErr(t *testing.T, d deps, in ResumeItemsInput) error {
	t.Helper()
	_, _, err := handleResumeItems(context.Background(), &mcp.CallToolRequest{}, in, d)
	return err
}

func intPtr(n int) *int { return &n }

// assertVectorStoreInSync checks chunks, text, metadata, and counts against the
// stored resume.
func assertVectorStoreInSync(t *testing.T, d deps) {
	t.Helper()
	stored, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load resume: %v", err)
	}
	data := stored.Data

	for i, e := range data.Experiences {
		for j, b := range e.Bullets {
			c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(i, j))
			if !ok {
				t.Fatalf("missing chunk for experience %d bullet %d", i, j)
			}
			if c.Text != b {
				t.Fatalf("experience %d bullet %d text = %q, want %q", i, j, c.Text, b)
			}
			if c.Metadata.Company != e.Company || c.Metadata.Role != e.Role {
				t.Fatalf("experience %d bullet %d metadata = %+v, want company %q role %q", i, j, c.Metadata, e.Company, e.Role)
			}
		}
	}
	for i, p := range data.Projects {
		for j, b := range p.Bullets {
			c, ok := d.VectorStore.GetChunk(vectorstore.ProjectChunkID(i, j))
			if !ok {
				t.Fatalf("missing chunk for project %d bullet %d", i, j)
			}
			if c.Text != b {
				t.Fatalf("project %d bullet %d text = %q, want %q", i, j, c.Text, b)
			}
		}
	}
	for i, s := range data.Skills {
		c, ok := d.VectorStore.GetChunk(vectorstore.SkillChunkID(i))
		if !ok || c.Metadata.Category != s.Category {
			t.Fatalf("skill %d chunk = %+v, %v; want category %q", i, c, ok, s.Category)
		}
	}
	for i, e := range data.Education {
		c, ok := d.VectorStore.GetChunk(vectorstore.EducationChunkID(i))
		if !ok || c.Metadata.Institution != e.Institution {
			t.Fatalf("education %d chunk = %+v, %v; want institution %q", i, c, ok, e.Institution)
		}
	}
	if got, want := d.VectorStore.ChunkCount(), countChunks(data); got != want {
		t.Fatalf("vector chunk count = %d, want %d", got, want)
	}
}

func bestBullet(result *vectorstore.SearchResult) string {
	best := ""
	bestScore := -1.0
	for _, exp := range result.Experiences {
		for _, b := range exp.Bullets {
			if b.Score > bestScore {
				bestScore = b.Score
				best = b.Text
			}
		}
	}
	for _, proj := range result.Projects {
		for _, b := range proj.Bullets {
			if b.Score > bestScore {
				bestScore = b.Score
				best = b.Text
			}
		}
	}
	return best
}

// --- health tool ---

func TestHealthWithoutResume(t *testing.T) {
	d := crudTestDeps(t)
	_, out, err := handleHealth(d)
	if err != nil {
		t.Fatalf("handleHealth: %v", err)
	}
	if out.HasResume {
		t.Error("HasResume = true, want false")
	}
	if out.HasEmbeddingConfig {
		t.Error("HasEmbeddingConfig = true, want false")
	}
	if out.VectorChunkCount != 0 {
		t.Errorf("VectorChunkCount = %d, want 0", out.VectorChunkCount)
	}
	if out.ResumeStats != (resume.Stats{}) {
		t.Errorf("ResumeStats = %+v, want zero", out.ResumeStats)
	}
	if out.InitializedAt != "" {
		t.Errorf("InitializedAt = %q, want empty", out.InitializedAt)
	}
}

func TestHealthAfterInit(t *testing.T) {
	d := initWithMockEmbedding(t)
	_, out, err := handleHealth(d)
	if err != nil {
		t.Fatalf("handleHealth: %v", err)
	}
	if !out.HasResume || !out.HasEmbeddingConfig {
		t.Fatalf("health = %+v, want resume and config present", out)
	}
	if want := resume.ComputeStats(sampleResumeData()); out.ResumeStats != want {
		t.Errorf("ResumeStats = %+v, want %+v", out.ResumeStats, want)
	}
	if want := countChunks(sampleResumeData()); out.VectorChunkCount != want {
		t.Errorf("VectorChunkCount = %d, want %d", out.VectorChunkCount, want)
	}
	if out.InitializedAt == "" {
		t.Error("InitializedAt is empty, want timestamp")
	}
}

// --- get ---

func TestItemsGetAllAndFilters(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{Operation: "get"})
	if len(out.Items) != 6 {
		t.Fatalf("get all returned %d items, want 6", len(out.Items))
	}
	byType := map[string]int{}
	for _, it := range out.Items {
		byType[it.Type]++
	}
	want := map[string]int{
		resume.ItemExperience: 2,
		resume.ItemSkill:      2,
		resume.ItemProject:    1,
		resume.ItemEducation:  1,
	}
	if !reflect.DeepEqual(byType, want) {
		t.Fatalf("type counts = %v, want %v", byType, want)
	}

	exps := callItems(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemExperience})
	if len(exps.Items) != 2 || exps.Items[0].Experience.Company != "E2E Corp" {
		t.Fatalf("experience filter = %+v", exps.Items)
	}
	if exps.Items[1].Experience.Company != "Previous Co" {
		t.Fatalf("second experience company = %q, want Previous Co", exps.Items[1].Experience.Company)
	}

	one := callItems(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemExperience, ID: intPtr(0)})
	if len(one.Items) != 1 || one.Items[0].ID != 0 || one.Items[0].Experience.Role != "Senior Engineer" {
		t.Fatalf("get by id = %+v", one.Items)
	}

	skills := callItems(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemSkill})
	if len(skills.Items) != 2 || skills.Items[1].Skill.Category != "Cloud" {
		t.Fatalf("skill filter = %+v", skills.Items)
	}

	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "get", Type: "bogus"}); err == nil || !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("invalid type error = %v, want invalid type", err)
	}
	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "get", ID: intPtr(0)}); err == nil || !strings.Contains(err.Error(), "type is required") {
		t.Fatalf("id without type error = %v, want type is required", err)
	}
}

func TestItemsGetBullets(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "get", Type: resume.ItemBullet,
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
	})
	if len(out.Items) != 3 {
		t.Fatalf("bullets = %d, want 3", len(out.Items))
	}
	if out.Items[0].ID != 0 || *out.Items[0].ParentID != 0 || out.Items[0].Type != resume.ItemBullet {
		t.Fatalf("first bullet entry = %+v", out.Items[0])
	}
	if !strings.Contains(out.Items[1].Bullet, "Led team of 5") {
		t.Fatalf("second bullet = %q", out.Items[1].Bullet)
	}

	one := callItems(t, d, ResumeItemsInput{
		Operation: "get", Type: resume.ItemBullet, ID: intPtr(2),
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
	})
	if len(one.Items) != 1 || !strings.Contains(one.Items[0].Bullet, "Reduced infrastructure costs") {
		t.Fatalf("single bullet = %+v", one.Items)
	}

	proj := callItems(t, d, ResumeItemsInput{
		Operation: "get", Type: resume.ItemBullet,
		ParentType: resume.ParentProject, ParentID: intPtr(0),
	})
	if len(proj.Items) != 1 || proj.Items[0].ParentType != resume.ParentProject {
		t.Fatalf("project bullets = %+v", proj.Items)
	}

	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemBullet, ParentType: resume.ParentExperience}); err == nil || !strings.Contains(err.Error(), "parentId is required") {
		t.Fatalf("missing parentId error = %v", err)
	}
	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemBullet, ParentID: intPtr(0)}); err == nil || !strings.Contains(err.Error(), "parentType is required") {
		t.Fatalf("missing parentType error = %v", err)
	}
	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemBullet, ParentType: "skill", ParentID: intPtr(0)}); err == nil || !strings.Contains(err.Error(), "invalid parentType") {
		t.Fatalf("bad parentType error = %v", err)
	}
}

// --- validation ---

func TestItemsInvalidOperations(t *testing.T) {
	d := initWithMockEmbedding(t)

	cases := []struct {
		name string
		in   ResumeItemsInput
		want string
	}{
		{"empty operation", ResumeItemsInput{}, "operation is required"},
		{"unknown operation", ResumeItemsInput{Operation: "purge"}, "invalid operation"},
		{"add without item", ResumeItemsInput{Operation: "add", Type: resume.ItemExperience}, "item is required"},
		{"add invalid type", ResumeItemsInput{Operation: "add", Type: "bogus", Item: &ItemInput{}}, "invalid type"},
		{"update without item", ResumeItemsInput{Operation: "update", Type: resume.ItemExperience, ID: intPtr(0)}, "item is required"},
		{"update without id", ResumeItemsInput{Operation: "update", Type: resume.ItemExperience, Item: &ItemInput{Role: "X"}}, "id is required"},
		{"delete without id", ResumeItemsInput{Operation: "delete", Type: resume.ItemExperience}, "id is required"},
		{"delete invalid type", ResumeItemsInput{Operation: "delete", Type: "bogus", ID: intPtr(0)}, "invalid type"},
		{"search without query", ResumeItemsInput{Operation: "search"}, "query is required"},
		{"update id out of range", ResumeItemsInput{Operation: "update", Type: resume.ItemExperience, ID: intPtr(99), Item: &ItemInput{Role: "X"}}, "out of range"},
	}
	for _, tc := range cases {
		err := callItemsErr(t, d, tc.in)
		if err == nil {
			t.Errorf("%s: expected error", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s error = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestItemsOnMissingResume(t *testing.T) {
	d := crudTestDeps(t)

	cases := []ResumeItemsInput{
		{Operation: "get"},
		{Operation: "add", Type: resume.ItemSkill, Item: &ItemInput{Category: "X", Values: "Y"}},
		{Operation: "delete", Type: resume.ItemSkill, ID: intPtr(0)},
	}
	for _, in := range cases {
		err := callItemsErr(t, d, in)
		if err == nil {
			t.Errorf("%s/%s: expected error", in.Operation, in.Type)
			continue
		}
		if !strings.Contains(err.Error(), "init") {
			t.Errorf("%s/%s error = %v, want init hint", in.Operation, in.Type, err)
		}
	}

	// Search on a fresh store fails before reaching the resume, because no
	// embedding config exists yet either.
	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "search", Query: "whatever"}); err == nil || !strings.Contains(err.Error(), "embedding config") {
		t.Fatalf("search on missing store error = %v, want embedding config error", err)
	}
}
