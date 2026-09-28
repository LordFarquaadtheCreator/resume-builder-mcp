package mcpserver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// --- init operation ---

func TestItemsInitCreatesResume(t *testing.T) {
	d := crudTestDeps(t)
	setMockConfig(t, d, mockEmbedServer(t))

	data := sampleResumeData()
	out := initResumeForTest(t, d, data)

	if out.Operation != "init" {
		t.Fatalf("operation = %q, want init", out.Operation)
	}
	if out.Stats == nil || *out.Stats != resume.ComputeStats(data) {
		t.Fatalf("stats = %+v, want %+v", out.Stats, resume.ComputeStats(data))
	}
	if out.VectorChunks == nil || *out.VectorChunks != countChunks(data) {
		t.Fatalf("vectorChunks = %v, want %d", out.VectorChunks, countChunks(data))
	}
	if !d.ResumeStore.Exists() {
		t.Fatal("resume file missing after init")
	}
	for _, c := range d.VectorStore.Snapshot() {
		if c.Embedding == nil {
			t.Fatalf("chunk %s has no embedding after init", c.ID)
		}
	}
	assertVectorStoreInSync(t, d)

	_, h, err := handleHealth(d)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !h.HasResume || h.InitializedAt == "" {
		t.Fatalf("health after init = %+v", h)
	}
}

func TestItemsInitRequiresDataAndName(t *testing.T) {
	d := crudTestDeps(t)

	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "init"}); err == nil || !strings.Contains(err.Error(), "data is required") {
		t.Fatalf("init without data error = %v", err)
	}
	empty := resume.ResumeData{}
	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "init", Data: &empty}); err == nil || !strings.Contains(err.Error(), "data.name") {
		t.Fatalf("init without name error = %v", err)
	}
	if d.ResumeStore.Exists() {
		t.Fatal("failed init created a resume file")
	}
	if d.VectorStore.ChunkCount() != 0 {
		t.Fatal("failed init wrote vector store chunks")
	}
}

func TestItemsInitOverwritesExisting(t *testing.T) {
	d := initWithMockEmbedding(t)

	replacement := resume.ResumeData{
		Name:   "New User",
		Skills: []resume.SkillGroup{{Category: "Cloud", Values: "AWS"}},
	}
	out := initResumeForTest(t, d, replacement)

	if out.Stats.Experiences != 0 || out.Stats.Skills != 1 || out.Stats.Bullets != 0 {
		t.Fatalf("replacement stats = %+v", out.Stats)
	}
	if out.VectorChunks == nil || *out.VectorChunks != 1 {
		t.Fatalf("vectorChunks = %v, want 1 (full rebuild)", out.VectorChunks)
	}
	assertVectorStoreInSync(t, d)

	if _, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 0)); ok {
		t.Fatal("old experience chunk survived init overwrite")
	}
	stored, err := d.ResumeStore.Load()
	if err != nil || stored.Data.Name != "New User" {
		t.Fatalf("stored resume = %+v, %v", stored, err)
	}
}

func TestItemsInitEmbeddingFailureLeavesStateUntouched(t *testing.T) {
	d, dir := crudTestDepsAt(t)
	setMockConfig(t, d, mockEmbedServer(t))
	initResumeForTest(t, d, sampleResumeData())
	removeEmbeddingConfig(t, dir)

	before, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	beforeChunks := d.VectorStore.ChunkCount()

	replacement := resume.ResumeData{Name: "Should Not Apply", Skills: []resume.SkillGroup{{Category: "X", Values: "Y"}}}
	err = callItemsErr(t, d, ResumeItemsInput{Operation: "init", Data: &replacement})
	if err == nil || !strings.Contains(err.Error(), "embedding config") {
		t.Fatalf("init without config error = %v, want embedding config error", err)
	}

	after, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load after failure: %v", err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("init changed resume data despite failed embedding")
	}
	if d.VectorStore.ChunkCount() != beforeChunks {
		t.Fatalf("chunk count = %d, want %d", d.VectorStore.ChunkCount(), beforeChunks)
	}
}

// --- batch operation ---

func TestItemsBatchAppliesMixedOperations(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "batch",
		Requests: []BatchRequest{
			{Operation: "add", Type: resume.ItemExperience, Item: &ItemInput{
				Company: "Batch Corp", Role: "Engineer", Start: "Mar. 2026", End: "Present",
				Bullets: []string{"Shipped batch workflows."},
			}},
			{Operation: "add", Type: resume.ItemBullet, ParentType: resume.ParentExperience, ParentID: intPtr(2), Item: &ItemInput{Text: "Automated the release train."}},
			{Operation: "update", Type: resume.ItemSkill, ID: intPtr(0), Item: &ItemInput{Values: "Go, TypeScript, Python, Rust"}},
			{Operation: "delete", Type: resume.ItemEducation, ID: intPtr(0)},
		},
	})

	if out.Operation != "batch" || len(out.Results) != 4 {
		t.Fatalf("batch result = %+v, want 4 results", out)
	}
	if out.Results[0].Experience == nil || out.Results[0].Experience.Company != "Batch Corp" {
		t.Fatalf("result 0 = %+v", out.Results[0])
	}
	if out.Results[1].Type != resume.ItemBullet || out.Results[1].ID != 1 {
		t.Fatalf("result 1 = %+v", out.Results[1])
	}
	if out.Results[3].Education == nil || out.Results[3].Education.Institution != "Test University" {
		t.Fatalf("result 3 = %+v", out.Results[3])
	}
	if out.Stats == nil {
		t.Fatal("missing stats")
	}
	want := resume.Stats{Experiences: 3, Bullets: 8, Skills: 2, Projects: 1, Education: 0}
	if *out.Stats != want {
		t.Fatalf("stats = %+v, want %+v", *out.Stats, want)
	}
	assertVectorStoreInSync(t, d)

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "release train"})
	if got := bestBullet(res.Result); !strings.Contains(got, "release train") {
		t.Fatalf("best bullet = %q, want batch-added bullet", got)
	}
}

func TestItemsBatchIsAtomicOnError(t *testing.T) {
	d := initWithMockEmbedding(t)
	before, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	beforeChunks := d.VectorStore.ChunkCount()

	err = callItemsErr(t, d, ResumeItemsInput{
		Operation: "batch",
		Requests: []BatchRequest{
			{Operation: "add", Type: resume.ItemSkill, Item: &ItemInput{Category: "Databases", Values: "Postgres"}},
			{Operation: "delete", Type: resume.ItemExperience, ID: intPtr(99)},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "request 1") {
		t.Fatalf("batch error = %v, want request 1 failure", err)
	}

	after, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load after failed batch: %v", err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("failed batch persisted partial resume changes")
	}
	if d.VectorStore.ChunkCount() != beforeChunks {
		t.Fatalf("chunk count = %d, want %d after failed batch", d.VectorStore.ChunkCount(), beforeChunks)
	}
}

func TestItemsBatchEmbeddingFailureRollsBack(t *testing.T) {
	d, dir := crudTestDepsAt(t)
	setMockConfig(t, d, mockEmbedServer(t))
	initResumeForTest(t, d, sampleResumeData())
	removeEmbeddingConfig(t, dir)

	before, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	beforeChunks := d.VectorStore.ChunkCount()

	err = callItemsErr(t, d, ResumeItemsInput{
		Operation: "batch",
		Requests: []BatchRequest{
			{Operation: "add", Type: resume.ItemSkill, Item: &ItemInput{Category: "Databases", Values: "Postgres"}},
			{Operation: "update", Type: resume.ItemSkill, ID: intPtr(0), Item: &ItemInput{Values: "Go, Rust"}},
			{Operation: "delete", Type: resume.ItemProject, ID: intPtr(0)},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "embedding config") {
		t.Fatalf("batch without config error = %v, want embedding config error", err)
	}

	after, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load after failed batch: %v", err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("batch changed resume data despite failed embedding")
	}
	if d.VectorStore.ChunkCount() != beforeChunks {
		t.Fatalf("chunk count = %d, want %d", d.VectorStore.ChunkCount(), beforeChunks)
	}
}

func TestItemsBatchRenumbersWithinBatch(t *testing.T) {
	d := initWithMockEmbedding(t)

	// Second request must see the state after the first: deleting experience 0
	// renumbers "Previous Co" to index 0.
	out := callItems(t, d, ResumeItemsInput{
		Operation: "batch",
		Requests: []BatchRequest{
			{Operation: "delete", Type: resume.ItemExperience, ID: intPtr(0)},
			{Operation: "add", Type: resume.ItemBullet, ParentType: resume.ParentExperience, ParentID: intPtr(0), Item: &ItemInput{Text: "Added after renumber."}},
		},
	})
	if len(out.Results) != 2 {
		t.Fatalf("results = %+v", out.Results)
	}

	b, err := d.ResumeStore.GetBullet(resume.ParentExperience, 0, 2)
	if err != nil || b != "Added after renumber." {
		t.Fatalf("bullet after batch = %q, %v", b, err)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 0))
	if !ok || !strings.Contains(c.Text, "React frontend") {
		t.Fatalf("renumbered chunk = %+v, %v; want Previous Co bullet", c, ok)
	}
	assertVectorStoreInSync(t, d)
}

func TestItemsBatchValidation(t *testing.T) {
	d := initWithMockEmbedding(t)

	if err := callItemsErr(t, d, ResumeItemsInput{Operation: "batch"}); err == nil || !strings.Contains(err.Error(), "requests is required") {
		t.Fatalf("empty batch error = %v", err)
	}
	err := callItemsErr(t, d, ResumeItemsInput{Operation: "batch", Requests: []BatchRequest{{Operation: "search"}}})
	if err == nil || !strings.Contains(err.Error(), "batch supports") {
		t.Fatalf("unsupported batch op error = %v", err)
	}
	err = callItemsErr(t, d, ResumeItemsInput{Operation: "batch", Requests: []BatchRequest{{Operation: "add", Type: resume.ItemSkill}}})
	if err == nil || !strings.Contains(err.Error(), "item is required") {
		t.Fatalf("batch add without item error = %v", err)
	}

	fresh := crudTestDeps(t)
	err = callItemsErr(t, fresh, ResumeItemsInput{
		Operation: "batch",
		Requests:  []BatchRequest{{Operation: "delete", Type: resume.ItemSkill, ID: intPtr(0)}},
	})
	if err == nil || !strings.Contains(err.Error(), "init") {
		t.Fatalf("batch on missing resume error = %v, want init hint", err)
	}
}
