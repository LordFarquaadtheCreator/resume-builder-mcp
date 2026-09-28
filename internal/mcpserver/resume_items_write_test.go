package mcpserver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// --- add ---

func TestItemsAddExperienceAndSearch(t *testing.T) {
	d := initWithMockEmbedding(t)
	before := countChunks(sampleResumeData())

	out := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemExperience,
		Item: &ItemInput{
			Company: "Kube Corp", Role: "Platform Engineer",
			Start: "Jan. 2026", End: "Present", Location: "Remote",
			Bullets: []string{"Managed Kubernetes clusters with Helm.", "Automated deployment pipelines."},
		},
	})
	if out.Item == nil || out.Item.ID != 2 || out.Item.Experience.Company != "Kube Corp" {
		t.Fatalf("add result = %+v", out.Item)
	}
	if out.Stats == nil || out.Stats.Experiences != 3 {
		t.Fatalf("stats = %+v, want 3 experiences", out.Stats)
	}
	if out.VectorChunks == nil || *out.VectorChunks != before+2 {
		t.Fatalf("vectorChunks = %v, want %d", out.VectorChunks, before+2)
	}
	assertVectorStoreInSync(t, d)

	e, err := d.ResumeStore.GetExperience(2)
	if err != nil || e.Company != "Kube Corp" {
		t.Fatalf("persisted experience = %+v, %v", e, err)
	}

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "kubernetes helm"})
	if len(res.Result.Experiences) == 0 {
		t.Fatal("search returned no experiences")
	}
	first := res.Result.Experiences[0]
	if first.Company != "Kube Corp" {
		t.Fatalf("first experience = %q, want Kube Corp", first.Company)
	}
	if !strings.Contains(first.Bullets[0].Text, "Kubernetes") {
		t.Fatalf("top bullet = %q, want Kubernetes bullet", first.Bullets[0].Text)
	}
}

func TestItemsAddBullet(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemBullet,
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
		Item: &ItemInput{Text: "Introduced platform SLOs."},
	})
	if out.Item == nil || out.Item.ID != 3 || *out.Item.ParentID != 0 {
		t.Fatalf("add bullet result = %+v", out.Item)
	}
	if out.Stats == nil || out.Stats.Bullets != 7 {
		t.Fatalf("bullets = %+v, want 7", out.Stats)
	}
	assertVectorStoreInSync(t, d)

	b, err := d.ResumeStore.GetBullet(resume.ParentExperience, 0, 3)
	if err != nil || b != "Introduced platform SLOs." {
		t.Fatalf("persisted bullet = %q, %v", b, err)
	}

	if err := callItemsErr(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemBullet,
		ParentType: resume.ParentExperience, ParentID: intPtr(99),
		Item: &ItemInput{Text: "x"},
	}); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("bad parent error = %v", err)
	}
	if err := callItemsErr(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemBullet,
		ParentType: "education", ParentID: intPtr(0),
		Item: &ItemInput{Text: "x"},
	}); err == nil || !strings.Contains(err.Error(), "invalid parentType") {
		t.Fatalf("bad parentType error = %v", err)
	}
	if err := callItemsErr(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemBullet,
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
		Item: &ItemInput{Text: "   "},
	}); err == nil || !strings.Contains(err.Error(), "item.text is required") {
		t.Fatalf("empty text error = %v", err)
	}
}

func TestItemsAddProjectSkillEducation(t *testing.T) {
	d := initWithMockEmbedding(t)

	proj := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemProject,
		Item: &ItemInput{Name: "Metrics Dash", Tech: "Go", Date: "Feb. 2025", Bullets: []string{"Built Grafana dashboards."}},
	})
	if proj.Item == nil || proj.Item.ID != 1 || proj.Item.Project.Name != "Metrics Dash" {
		t.Fatalf("add project = %+v", proj.Item)
	}

	skill := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemSkill,
		Item: &ItemInput{Category: "Databases", Values: "Postgres, Redis"},
	})
	if skill.Item == nil || skill.Item.ID != 2 || skill.Item.Skill.Category != "Databases" {
		t.Fatalf("add skill = %+v", skill.Item)
	}

	edu := callItems(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemEducation,
		Item: &ItemInput{Institution: "State U", Degree: "M.S. Computer Science"},
	})
	if edu.Item == nil || edu.Item.ID != 1 || edu.Item.Education.Institution != "State U" {
		t.Fatalf("add education = %+v", edu.Item)
	}

	assertVectorStoreInSync(t, d)

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "postgres redis"})
	if len(res.Result.Skills) == 0 || res.Result.Skills[0].Chunk.Metadata.Category != "Databases" {
		t.Fatalf("skill search = %+v, want Databases first", res.Result.Skills)
	}
}

func TestItemsAddWithoutConfigRollsBack(t *testing.T) {
	d, dir := crudTestDepsAt(t)
	setMockConfig(t, d, mockEmbedServer(t))
	initResumeForTest(t, d, sampleResumeData())
	removeEmbeddingConfig(t, dir)

	before, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load resume: %v", err)
	}
	beforeChunks := d.VectorStore.ChunkCount()

	err = callItemsErr(t, d, ResumeItemsInput{
		Operation: "add", Type: resume.ItemExperience,
		Item: &ItemInput{Company: "No Config Co", Bullets: []string{"Should not be indexed."}},
	})
	if err == nil || !strings.Contains(err.Error(), "embedding config") {
		t.Fatalf("add without config error = %v, want embedding config error", err)
	}

	after, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load resume after failure: %v", err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("resume data changed despite failed embedding (rollback broken)")
	}
	if d.VectorStore.ChunkCount() != beforeChunks {
		t.Fatalf("vector chunk count = %d, want %d (rollback broken)", d.VectorStore.ChunkCount(), beforeChunks)
	}
}

// --- update ---

func TestItemsUpdateExperiencePartialAndBullets(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "update", Type: resume.ItemExperience, ID: intPtr(1),
		Item: &ItemInput{Role: "Senior Software Engineer"},
	})
	e := out.Item.Experience
	if e.Company != "Previous Co" || e.Role != "Senior Software Engineer" || len(e.Bullets) != 2 {
		t.Fatalf("partial update = %+v, want merged fields", e)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(1, 0))
	if !ok || c.Metadata.Role != "Senior Software Engineer" || c.Text != "Developed React frontend with TypeScript." {
		t.Fatalf("chunk after metadata update = %+v, %v", c, ok)
	}
	assertVectorStoreInSync(t, d)

	out = callItems(t, d, ResumeItemsInput{
		Operation: "update", Type: resume.ItemExperience, ID: intPtr(1),
		Item: &ItemInput{Bullets: []string{"Only bullet now."}},
	})
	e = out.Item.Experience
	if e.Role != "Senior Software Engineer" || len(e.Bullets) != 1 || e.Bullets[0] != "Only bullet now." {
		t.Fatalf("bullets update = %+v, want replaced bullets and kept role", e)
	}
	if _, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(1, 1)); ok {
		t.Fatal("stale bullet chunk still present after bullet list shrink")
	}
	c, _ = d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(1, 0))
	if c.Text != "Only bullet now." {
		t.Fatalf("chunk text = %q, want Only bullet now.", c.Text)
	}
	assertVectorStoreInSync(t, d)
}

func TestItemsUpdateBulletChangesRanking(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "update", Type: resume.ItemBullet, ID: intPtr(0),
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
		Item: &ItemInput{Text: "Built Kubernetes autoscaling platform."},
	})
	if out.Item.Bullet != "Built Kubernetes autoscaling platform." {
		t.Fatalf("updated bullet = %+v", out.Item)
	}
	b, err := d.ResumeStore.GetBullet(resume.ParentExperience, 0, 0)
	if err != nil || b != "Built Kubernetes autoscaling platform." {
		t.Fatalf("persisted bullet = %q, %v", b, err)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 0))
	if !ok || c.Text != "Built Kubernetes autoscaling platform." {
		t.Fatalf("chunk after update = %+v, %v", c, ok)
	}
	assertVectorStoreInSync(t, d)

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "kubernetes autoscaling"})
	if got := bestBullet(res.Result); !strings.Contains(got, "autoscaling") {
		t.Fatalf("best bullet = %q, want updated bullet to rank first", got)
	}
}

func TestItemsUpdateMetadataOnlyWithoutConfig(t *testing.T) {
	d, dir := crudTestDepsAt(t)
	setMockConfig(t, d, mockEmbedServer(t))
	initResumeForTest(t, d, sampleResumeData())
	removeEmbeddingConfig(t, dir)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "update", Type: resume.ItemExperience, ID: intPtr(0),
		Item: &ItemInput{Location: "Remote (US)"},
	})
	if out.Item.Experience.Location != "Remote (US)" {
		t.Fatalf("location = %q, want Remote (US)", out.Item.Experience.Location)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 0))
	if !ok || c.Metadata.Location != "Remote (US)" {
		t.Fatalf("chunk metadata after config removal = %+v, %v", c, ok)
	}
	assertVectorStoreInSync(t, d)
}

// --- delete ---

func TestItemsDeleteExperienceRenumbers(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{Operation: "delete", Type: resume.ItemExperience, ID: intPtr(0)})
	if out.Item == nil || out.Item.Experience.Company != "E2E Corp" {
		t.Fatalf("delete result = %+v", out.Item)
	}
	stored, err := d.ResumeStore.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(stored.Data.Experiences) != 1 || stored.Data.Experiences[0].Company != "Previous Co" {
		t.Fatalf("experiences after delete = %+v", stored.Data.Experiences)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 0))
	if !ok || c.Text != "Developed React frontend with TypeScript." || c.Metadata.Company != "Previous Co" {
		t.Fatalf("renumbered chunk = %+v, %v", c, ok)
	}
	if _, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(2, 0)); ok {
		t.Fatal("stale chunk from the deleted experience still present")
	}
	assertVectorStoreInSync(t, d)

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "infrastructure costs optimization"})
	for _, exp := range res.Result.Experiences {
		if exp.Company == "E2E Corp" {
			t.Fatal("deleted experience still returned by search")
		}
	}
}

func TestItemsDeleteProjectAndSkill(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{Operation: "delete", Type: resume.ItemProject, ID: intPtr(0)})
	if out.Item == nil || out.Item.Project.Name != "E2E Tool" {
		t.Fatalf("delete project result = %+v", out.Item)
	}
	assertVectorStoreInSync(t, d)

	res := callItems(t, d, ResumeItemsInput{Operation: "search", Query: "CLI tool automating deployment workflows"})
	if len(res.Result.Projects) != 0 {
		t.Fatalf("projects after delete = %+v, want none", res.Result.Projects)
	}

	out = callItems(t, d, ResumeItemsInput{Operation: "delete", Type: resume.ItemSkill, ID: intPtr(1)})
	if out.Item == nil || out.Item.Skill.Category != "Cloud" {
		t.Fatalf("delete skill result = %+v", out.Item)
	}
	skills := callItems(t, d, ResumeItemsInput{Operation: "get", Type: resume.ItemSkill})
	if len(skills.Items) != 1 || skills.Items[0].Skill.Category != "Languages" {
		t.Fatalf("skills after delete = %+v", skills.Items)
	}
	assertVectorStoreInSync(t, d)
}

func TestItemsDeleteBullet(t *testing.T) {
	d := initWithMockEmbedding(t)

	out := callItems(t, d, ResumeItemsInput{
		Operation: "delete", Type: resume.ItemBullet, ID: intPtr(1),
		ParentType: resume.ParentExperience, ParentID: intPtr(0),
	})
	if out.Item == nil || !strings.Contains(out.Item.Bullet, "Led team of 5") {
		t.Fatalf("delete bullet result = %+v", out.Item)
	}
	b, err := d.ResumeStore.GetBullet(resume.ParentExperience, 0, 1)
	if err != nil || !strings.Contains(b, "Reduced infrastructure costs") {
		t.Fatalf("bullet after delete = %q, %v (want renumbered)", b, err)
	}
	c, ok := d.VectorStore.GetChunk(vectorstore.ExperienceChunkID(0, 1))
	if !ok || !strings.Contains(c.Text, "Reduced infrastructure costs") {
		t.Fatalf("renumbered bullet chunk = %+v, %v", c, ok)
	}
	assertVectorStoreInSync(t, d)
}
