package vectorstore

import (
	"reflect"
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

func TestChunkCRUD(t *testing.T) {
	s := NewStore(t.TempDir())
	c := Chunk{ID: ExperienceChunkID(0, 0), Type: ChunkExperienceBullet, Text: "hello", Embedding: []float64{1, 0}}

	if err := s.AddChunk(c); err != nil {
		t.Fatalf("AddChunk: %v", err)
	}
	if err := s.AddChunk(c); err == nil {
		t.Fatal("AddChunk duplicate: expected error")
	}

	got, ok := s.GetChunk(c.ID)
	if !ok || got.Text != "hello" {
		t.Fatalf("GetChunk = %+v, %v; want hello, true", got, ok)
	}

	updated := Chunk{ID: "overwritten-by-update", Type: ChunkExperienceBullet, Text: "bye", Embedding: []float64{0, 1}}
	if err := s.UpdateChunk(c.ID, updated); err != nil {
		t.Fatalf("UpdateChunk: %v", err)
	}
	got, _ = s.GetChunk(c.ID)
	if got.Text != "bye" || !reflect.DeepEqual(got.Embedding, []float64{0, 1}) {
		t.Fatalf("after update = %+v, want bye with new embedding", got)
	}
	if err := s.UpdateChunk("missing", Chunk{}); err == nil {
		t.Fatal("UpdateChunk missing: expected error")
	}

	if err := s.DeleteChunk(c.ID); err != nil {
		t.Fatalf("DeleteChunk: %v", err)
	}
	if _, ok := s.GetChunk(c.ID); ok {
		t.Fatal("chunk still present after delete")
	}
	if err := s.DeleteChunk(c.ID); err == nil {
		t.Fatal("DeleteChunk missing: expected error")
	}
	if s.ChunkCount() != 0 {
		t.Fatalf("ChunkCount = %d, want 0", s.ChunkCount())
	}
}

func TestSyncChunksKeepsEmbeddingForUnchangedText(t *testing.T) {
	s := NewStore(t.TempDir())
	exp := resume.Experience{Company: "Acme", Role: "Engineer"}
	id := ExperienceChunkID(0, 0)

	chunk := ExperienceBulletChunk(exp, 0, 0, "did things")
	s.SyncChunks([]Chunk{chunk}, map[string][]float64{id: {1, 0, 0}}, nil)

	// Metadata change only: no embedding supplied, the existing one is kept.
	exp.Company = "Acme 2"
	updated := ExperienceBulletChunk(exp, 0, 0, "did things")
	s.SyncChunks([]Chunk{updated}, nil, nil)

	got, ok := s.GetChunk(id)
	if !ok {
		t.Fatal("chunk missing after metadata sync")
	}
	if got.Metadata.Company != "Acme 2" {
		t.Fatalf("company = %q, want Acme 2", got.Metadata.Company)
	}
	if !reflect.DeepEqual(got.Embedding, []float64{1, 0, 0}) {
		t.Fatalf("embedding = %v, want unchanged [1 0 0]", got.Embedding)
	}

	// Text change: the new embedding is used.
	changed := ExperienceBulletChunk(exp, 0, 0, "did other things")
	s.SyncChunks([]Chunk{changed}, map[string][]float64{id: {0, 1, 0}}, nil)
	got, _ = s.GetChunk(id)
	if got.Text != "did other things" {
		t.Fatalf("text = %q, want did other things", got.Text)
	}
	if !reflect.DeepEqual(got.Embedding, []float64{0, 1, 0}) {
		t.Fatalf("embedding = %v, want [0 1 0]", got.Embedding)
	}
	if s.ChunkCount() != 1 {
		t.Fatalf("ChunkCount = %d, want 1", s.ChunkCount())
	}
}

func TestSyncChunksRemovesStaleIDs(t *testing.T) {
	s := NewStore(t.TempDir())
	a := ExperienceBulletChunk(resume.Experience{}, 0, 0, "a")
	b := ExperienceBulletChunk(resume.Experience{}, 0, 1, "b")
	s.SyncChunks([]Chunk{a, b}, map[string][]float64{a.ID: {1}, b.ID: {1}}, nil)

	s.SyncChunks([]Chunk{a}, nil, []string{b.ID})
	if s.ChunkCount() != 1 {
		t.Fatalf("ChunkCount = %d, want 1", s.ChunkCount())
	}
	if _, ok := s.GetChunk(b.ID); ok {
		t.Fatal("stale chunk still present")
	}
}

func TestNeedsReembed(t *testing.T) {
	s := NewStore(t.TempDir())
	id := EducationChunkID(0)
	if !s.NeedsReembed(id, "text") {
		t.Fatal("missing chunk should need embedding")
	}
	chunk := EducationChunk(resume.Education{Institution: "U", Degree: "B.S."}, 0)
	s.SyncChunks([]Chunk{chunk}, map[string][]float64{id: {1}}, nil)

	if s.NeedsReembed(id, chunk.Text) {
		t.Fatal("unchanged text should not need embedding")
	}
	if !s.NeedsReembed(id, "U - M.S.") {
		t.Fatal("changed text should need embedding")
	}
}

func TestDeleteExperienceRenumbersLaterChunks(t *testing.T) {
	s := NewStore(t.TempDir())
	seed := []Chunk{
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "a0"),
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 1, "a1"),
		ExperienceBulletChunk(resume.Experience{Company: "B"}, 1, 0, "b0"),
		SkillChunk(resume.SkillGroup{Category: "L", Values: "Go"}, 0),
		ProjectBulletChunk(resume.Project{Name: "P"}, 0, 0, "p0"),
		EducationChunk(resume.Education{Institution: "U", Degree: "D"}, 0),
	}
	embs := make(map[string][]float64, len(seed))
	for _, c := range seed {
		embs[c.ID] = []float64{1}
	}
	s.SyncChunks(seed, embs, nil)

	s.DeleteExperience(0)

	if s.ChunkCount() != 4 {
		t.Fatalf("ChunkCount = %d, want 4", s.ChunkCount())
	}
	c, ok := s.GetChunk("exp_0_bullet_0")
	if !ok || c.Text != "b0" || c.Metadata.Company != "B" {
		t.Fatalf("renumbered chunk = %+v, %v; want b0 from company B", c, ok)
	}
	if _, ok := s.GetChunk("exp_1_bullet_0"); ok {
		t.Fatal("stale experience chunk still present")
	}
	for _, id := range []string{"skill_0", "proj_0_bullet_0", "edu_0"} {
		if _, ok := s.GetChunk(id); !ok {
			t.Fatalf("chunk %s should be untouched", id)
		}
	}
}

func TestDeleteProjectRenumbersLaterChunks(t *testing.T) {
	s := NewStore(t.TempDir())
	seed := []Chunk{
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "a0"),
		ProjectBulletChunk(resume.Project{Name: "P1"}, 0, 0, "p1a"),
		ProjectBulletChunk(resume.Project{Name: "P1"}, 0, 1, "p1b"),
		ProjectBulletChunk(resume.Project{Name: "P2"}, 1, 0, "p2a"),
	}
	embs := make(map[string][]float64, len(seed))
	for _, c := range seed {
		embs[c.ID] = []float64{1}
	}
	s.SyncChunks(seed, embs, nil)

	s.DeleteProject(0)

	if s.ChunkCount() != 2 {
		t.Fatalf("ChunkCount = %d, want 2", s.ChunkCount())
	}
	c, ok := s.GetChunk("proj_0_bullet_0")
	if !ok || c.Text != "p2a" || c.Metadata.ProjectName != "P2" {
		t.Fatalf("renumbered chunk = %+v, %v; want p2a", c, ok)
	}
	if _, ok := s.GetChunk("exp_0_bullet_0"); !ok {
		t.Fatal("experience chunk should be untouched")
	}
}

func TestDeleteSkillAndEducationRenumber(t *testing.T) {
	s := NewStore(t.TempDir())
	seed := []Chunk{
		SkillChunk(resume.SkillGroup{Category: "Languages", Values: "Go"}, 0),
		SkillChunk(resume.SkillGroup{Category: "Cloud", Values: "AWS"}, 1),
		EducationChunk(resume.Education{Institution: "U1", Degree: "B.S."}, 0),
		EducationChunk(resume.Education{Institution: "U2", Degree: "M.S."}, 1),
	}
	embs := make(map[string][]float64, len(seed))
	for _, c := range seed {
		embs[c.ID] = []float64{1}
	}
	s.SyncChunks(seed, embs, nil)

	s.DeleteSkill(0)
	skill, ok := s.GetChunk("skill_0")
	if !ok || skill.Metadata.Category != "Cloud" {
		t.Fatalf("skill_0 = %+v, %v; want Cloud", skill, ok)
	}
	if _, ok := s.GetChunk("skill_1"); ok {
		t.Fatal("stale skill_1 still present")
	}

	s.DeleteEducation(0)
	edu, ok := s.GetChunk("edu_0")
	if !ok || edu.Metadata.Institution != "U2" {
		t.Fatalf("edu_0 = %+v, %v; want U2", edu, ok)
	}
	if _, ok := s.GetChunk("edu_1"); ok {
		t.Fatal("stale edu_1 still present")
	}
}

func TestDeleteBulletRenumbersWithinParent(t *testing.T) {
	s := NewStore(t.TempDir())
	seed := []Chunk{
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "b0"),
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 1, "b1"),
		ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 2, "b2"),
		ExperienceBulletChunk(resume.Experience{Company: "B"}, 1, 0, "other"),
	}
	embs := make(map[string][]float64, len(seed))
	for _, c := range seed {
		embs[c.ID] = []float64{1}
	}
	s.SyncChunks(seed, embs, nil)

	s.DeleteBulletFromParent(resume.ParentExperience, 0, 1)

	if s.ChunkCount() != 3 {
		t.Fatalf("ChunkCount = %d, want 3", s.ChunkCount())
	}
	c, ok := s.GetChunk("exp_0_bullet_1")
	if !ok || c.Text != "b2" {
		t.Fatalf("exp_0_bullet_1 = %+v, %v; want b2", c, ok)
	}
	if _, ok := s.GetChunk("exp_0_bullet_2"); ok {
		t.Fatal("stale bullet chunk still present")
	}
	if _, ok := s.GetChunk("exp_1_bullet_0"); !ok {
		t.Fatal("sibling experience should be untouched")
	}
}

func TestSyncChunkSliceEmbeddingSemantics(t *testing.T) {
	original := []Chunk{ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "old text")}
	original[0].Embedding = []float64{1, 0}

	// Unchanged text keeps its embedding, even when no embeddings are supplied.
	out := SyncChunkSlice(original, []Chunk{ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "old text")}, nil, nil)
	if !reflect.DeepEqual(out[0].Embedding, []float64{1, 0}) {
		t.Fatalf("embedding = %v, want carried over", out[0].Embedding)
	}

	// Changed text without a supplied embedding is left nil so the caller embeds it.
	out = SyncChunkSlice(original, []Chunk{ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "new text")}, nil, nil)
	if out[0].Embedding != nil {
		t.Fatalf("embedding = %v, want nil", out[0].Embedding)
	}

	// Supplied embeddings are used for changed text.
	out = SyncChunkSlice(original, []Chunk{ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "new text")}, map[string][]float64{original[0].ID: {0, 1}}, nil)
	if !reflect.DeepEqual(out[0].Embedding, []float64{0, 1}) {
		t.Fatalf("embedding = %v, want [0 1]", out[0].Embedding)
	}

	// The input slice is not mutated.
	if original[0].Text != "old text" || !reflect.DeepEqual(original[0].Embedding, []float64{1, 0}) {
		t.Fatalf("input slice mutated: %+v", original[0])
	}
}

func TestBuildChunks(t *testing.T) {
	data := resume.ResumeData{
		Experiences: []resume.Experience{{Company: "A", Bullets: []string{"a1", "a2"}}},
		Skills:      []resume.SkillGroup{{Category: "Languages", Values: "Go"}},
		Projects:    []resume.Project{{Name: "P", Bullets: []string{"p1"}}},
		Education:   []resume.Education{{Institution: "U", Degree: "D"}},
	}
	chunks := BuildChunks(data)
	if len(chunks) != 5 {
		t.Fatalf("chunks = %d, want 5", len(chunks))
	}
	if chunks[0].ID != ExperienceChunkID(0, 0) || chunks[0].Type != ChunkExperienceBullet {
		t.Fatalf("chunk[0] = %+v", chunks[0])
	}
	last := chunks[len(chunks)-1]
	if last.ID != EducationChunkID(0) || last.Type != ChunkEducation {
		t.Fatalf("last chunk = %+v", last)
	}
	for _, c := range chunks {
		if c.Embedding != nil {
			t.Fatalf("chunk %s has an embedding, want none", c.ID)
		}
	}
}

func TestSnapshotAndSetChunks(t *testing.T) {
	s := NewStore(t.TempDir())
	c := ExperienceBulletChunk(resume.Experience{}, 0, 0, "a")
	s.SetChunks([]Chunk{c})

	snap := s.Snapshot()
	snap[0].Text = "mutated"
	if got, _ := s.GetChunk(c.ID); got.Text != "a" {
		t.Fatalf("snapshot mutation leaked into store: %q", got.Text)
	}
}

func TestChunkIDCodecs(t *testing.T) {
	tests := []struct {
		id     string
		codec  chunkCodec
		parent int
		child  int
		ok     bool
	}{
		{ExperienceChunkID(3, 2), experienceCodec, 3, 2, true},
		{ProjectChunkID(0, 10), projectCodec, 0, 10, true},
		{SkillChunkID(4), skillCodec, 4, 0, true},
		{EducationChunkID(1), educationCodec, 1, 0, true},
		{"exp_x_bullet_0", experienceCodec, 0, 0, false},
		{"exp_1", experienceCodec, 0, 0, false},
		{"skill_", skillCodec, 0, 0, false},
		{"nope", experienceCodec, 0, 0, false},
		{"proj_0_bullet_0", experienceCodec, 0, 0, false},
	}
	for _, tc := range tests {
		parent, child, ok := tc.codec.parse(tc.id)
		if ok != tc.ok || (ok && (parent != tc.parent || child != tc.child)) {
			t.Errorf("parse(%q) = (%d, %d, %v), want (%d, %d, %v)", tc.id, parent, child, ok, tc.parent, tc.child, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if got := tc.codec.rebuild(parent, child); got != tc.id {
			t.Errorf("rebuild(%d, %d) = %q, want %q", parent, child, got, tc.id)
		}
	}
}

func TestCRUDPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	id := ExperienceChunkID(0, 0)
	chunk := ExperienceBulletChunk(resume.Experience{Company: "A"}, 0, 0, "a0")
	s.SyncChunks([]Chunk{chunk}, map[string][]float64{id: {1, 2, 3}}, nil)
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := NewStore(dir)
	if err := loaded.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := loaded.GetChunk(id)
	if !ok || got.Text != "a0" || !reflect.DeepEqual(got.Embedding, []float64{1, 2, 3}) {
		t.Fatalf("loaded chunk = %+v, %v; want a0 with embedding", got, ok)
	}

	loaded.DeleteExperience(0)
	if err := loaded.Save(); err != nil {
		t.Fatalf("save after delete: %v", err)
	}
	reloaded := NewStore(dir)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ChunkCount() != 0 {
		t.Fatalf("ChunkCount after delete round trip = %d, want 0", reloaded.ChunkCount())
	}
}
