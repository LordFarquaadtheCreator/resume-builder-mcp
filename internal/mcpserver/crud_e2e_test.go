package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// TestE2ECRUDWorkflow walks health → init → add → search → update → delete →
// generate, checking vector store sync after every mutation.
func TestE2ECRUDWorkflow(t *testing.T) {
	d := crudTestDeps(t)
	setMockConfig(t, d, mockEmbedServer(t))

	// 1. health before init
	_, h, err := handleHealth(d)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if h.HasResume {
		t.Fatal("HasResume = true before init")
	}

	// 2. init
	initOut := initResumeForTest(t, d, sampleResumeData())
	if initOut.Stats.Experiences != 2 || initOut.Stats.Bullets != 6 {
		t.Fatalf("init stats = %+v", initOut.Stats)
	}
	assertVectorStoreInSync(t, d)

	// 3. health after init
	_, h, err = handleHealth(d)
	if err != nil {
		t.Fatalf("health after init: %v", err)
	}
	if !h.HasResume || !h.HasEmbeddingConfig || h.VectorChunkCount != countChunks(sampleResumeData()) {
		t.Fatalf("health after init = %+v", h)
	}

	// 4. add experience
	addOut := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemExperience,
		Item: &ItemInput{
			Company: "Kube Corp", Role: "Platform Engineer",
			Start: "Feb. 2026", End: "Present",
			Bullets: []string{"Managed Kubernetes clusters with Helm.", "Cut deploy times by 70% with GitOps."},
		},
	})
	if addOut.Item.ID != 2 {
		t.Fatalf("added experience index = %d, want 2", addOut.Item.ID)
	}
	assertVectorStoreInSync(t, d)

	// 5. search finds the new item
	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "kubernetes helm gitops"})
	if len(res.Result.Experiences) == 0 || res.Result.Experiences[0].Company != "Kube Corp" {
		t.Fatalf("search after add = %+v", res.Result.Experiences)
	}

	// 6. update a bullet of the new experience
	callItems(t, d, ResumeItemsInput{
		Operation: "update", Type: resume.ItemBullet, ID: intPtr(0),
		ParentType: resume.ParentExperience, ParentID: intPtr(2),
		Item: &ItemInput{Text: "Orchestrated 50+ Kubernetes clusters with Helm and ArgoCD."},
	})
	assertVectorStoreInSync(t, d)

	// 7. delete the project; search no longer returns it
	callItems(t, d, ResumeItemsInput{Operation: "delete", Type: resume.ItemProject, ID: intPtr(0)})
	assertVectorStoreInSync(t, d)
	res = callItems(t, d, ResumeItemsInput{Operation: "search", Query: "CLI tool automating deployment"})
	if len(res.Result.Projects) != 0 {
		t.Fatalf("projects after delete = %+v, want none", res.Result.Projects)
	}

	// 8. delete an experience; later experiences renumber
	callItems(t, d, ResumeItemsInput{Operation: "delete", Type: resume.ItemExperience, ID: intPtr(1)})
	assertVectorStoreInSync(t, d)
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(1, 0))
	if !ok || c.Metadata.Company != "Kube Corp" {
		t.Fatalf("renumbered chunk after delete = %+v, %v; want Kube Corp", c, ok)
	}

	// 9. generate an auto-mode PDF from the mutated resume
	outDir := t.TempDir()
	_, genOut, err := handleGenerateResume(context.Background(), &mcp.CallToolRequest{}, GenerateResumeInput{
		Mode: "auto", Query: "kubernetes platform engineer", Template: "fahad", OutputDir: outDir,
	}, d)
	if err != nil {
		t.Fatalf("generate_resume: %v", err)
	}
	if pages := pdfPageCount(t, genOut.OutputPath); pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}
	if !genOut.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false")
	}
}

// TestE2EMultipleCRUDKeepsVectorStoreInSync runs a longer mutation sequence and
// verifies the store mirrors the resume after every step.
func TestE2EMultipleCRUDKeepsVectorStoreInSync(t *testing.T) {
	d := initWithMockEmbedding(t)

	ops := []ResumeItemsInput{
		{Operation: "add", Type: resume.ItemExperience, Item: &ItemInput{
			Company: "Side Quest", Role: "Consultant", Start: "Jan. 2021", End: "May 2022",
			Bullets: []string{"Advised teams on platform migrations."},
		}},
		{Operation: "add", Type: resume.ItemBullet, ParentType: resume.ParentExperience, ParentID: intPtr(0), Item: &ItemInput{Text: "Owned on-call rotation."}},
		{Operation: "add", Type: resume.ItemProject, Item: &ItemInput{Name: "Tiny Tool", Tech: "Rust", Bullets: []string{"Wrote a fast parser."}}},
		{Operation: "add", Type: resume.ItemSkill, Item: &ItemInput{Category: "Databases", Values: "Postgres, Redis"}},
		{Operation: "update", Type: resume.ItemExperience, ID: intPtr(0), Item: &ItemInput{Bullets: []string{"First.", "Second.", "Third."}}},
		{Operation: "update", Type: resume.ItemProject, ID: intPtr(0), Item: &ItemInput{Bullets: []string{"Rewrote the parser."}}},
		{Operation: "delete", Type: resume.ItemBullet, ID: intPtr(0), ParentType: resume.ParentExperience, ParentID: intPtr(0)},
		{Operation: "delete", Type: resume.ItemSkill, ID: intPtr(0)},
		{Operation: "delete", Type: resume.ItemEducation, ID: intPtr(0)},
		{Operation: "add", Type: resume.ItemEducation, Item: &ItemInput{Institution: "Night School", Degree: "Certificate"}},
		{Operation: "update", Type: resume.ItemBullet, ID: intPtr(0), ParentType: resume.ParentProject, ParentID: intPtr(0), Item: &ItemInput{Text: "Rewrote the parser in Rust."}},
		{Operation: "batch", Requests: []BatchRequest{
			{Operation: "add", Type: resume.ItemSkill, Item: &ItemInput{Category: "Testing", Values: "Go test"}},
			{Operation: "update", Type: resume.ItemBullet, ID: intPtr(0), ParentType: resume.ParentProject, ParentID: intPtr(0), Item: &ItemInput{Text: "Rewrote the parser in Rust, with tests."}},
			{Operation: "delete", Type: resume.ItemBullet, ID: intPtr(1), ParentType: resume.ParentExperience, ParentID: intPtr(0)},
		}},
		{Operation: "delete", Type: resume.ItemExperience, ID: intPtr(1)},
	}

	for i, op := range ops {
		callItems(t, d, op)
		assertVectorStoreInSync(t, d)
		t.Logf("op %d (%s %s) in sync", i, op.Operation, op.Type)
	}

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "rust parser"})
	if len(res.Result.Projects) == 0 {
		t.Fatal("search returned no projects after CRUD sequence")
	}
}
