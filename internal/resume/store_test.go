package resume

import (
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func sampleStoreData() ResumeData {
	return ResumeData{
		Name: "Test User",
		Education: []Education{
			{Institution: "Test U", Degree: "B.S. Computer Science"},
		},
		Skills: []SkillGroup{
			{Category: "Languages", Values: "Go, Python"},
		},
		Experiences: []Experience{
			{Company: "Acme", Role: "Engineer", Start: "Jan. 2024", End: "Present", Bullets: []string{"a1", "a2"}},
			{Company: "Old Co", Role: "Intern", Start: "Jun. 2022", End: "Dec. 2023", Bullets: []string{"b1"}},
		},
		Projects: []Project{
			{Name: "Tool", Tech: "Go", Bullets: []string{"p1"}},
		},
	}
}

func seededStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	if err := s.Save(sampleStoreData()); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	return s
}

func TestStoreExistsAndDelete(t *testing.T) {
	s := newTestStore(t)
	if s.Exists() {
		t.Fatal("Exists = true on empty store, want false")
	}
	if err := s.Save(sampleStoreData()); err != nil {
		t.Fatalf("save: %v", err)
	}
	if !s.Exists() {
		t.Fatal("Exists = false after save, want true")
	}
	if err := s.Delete(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if s.Exists() {
		t.Fatal("Exists = true after delete, want false")
	}
}

func TestCRUDWithoutResumeFails(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.AddExperience(Experience{Company: "X"}); err == nil || !strings.Contains(err.Error(), "init") {
		t.Fatalf("AddExperience error = %v, want init hint", err)
	}
	if _, err := s.AddBullet(ParentExperience, 0, "x"); err == nil {
		t.Fatal("AddBullet on missing resume: expected error")
	}
	if _, err := s.GetExperience(0); err == nil {
		t.Fatal("GetExperience on missing resume: expected error")
	}
}

func TestExperienceCRUD(t *testing.T) {
	s := seededStore(t)

	index, err := s.AddExperience(Experience{Company: "New Co", Role: "Dev", Bullets: []string{"n1"}})
	if err != nil {
		t.Fatalf("AddExperience: %v", err)
	}
	if index != 2 {
		t.Fatalf("index = %d, want 2", index)
	}

	e, err := s.GetExperience(index)
	if err != nil {
		t.Fatalf("GetExperience: %v", err)
	}
	if e.Company != "New Co" || len(e.Bullets) != 1 {
		t.Fatalf("GetExperience = %+v, want New Co with 1 bullet", e)
	}

	if err := s.UpdateExperience(index, Experience{Company: "New Co 2", Role: "Lead"}); err != nil {
		t.Fatalf("UpdateExperience: %v", err)
	}
	e, err = s.GetExperience(index)
	if err != nil {
		t.Fatalf("GetExperience after update: %v", err)
	}
	if e.Company != "New Co 2" || e.Role != "Lead" || len(e.Bullets) != 0 {
		t.Fatalf("updated experience = %+v, want replacement", e)
	}

	deleted, err := s.DeleteExperience(index)
	if err != nil {
		t.Fatalf("DeleteExperience: %v", err)
	}
	if deleted.Company != "New Co 2" {
		t.Fatalf("deleted = %+v, want New Co 2", deleted)
	}
	if _, err := s.GetExperience(index); err == nil {
		t.Fatal("GetExperience after delete: expected error")
	}
	stored, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(stored.Data.Experiences) != 2 {
		t.Fatalf("experiences = %d, want 2", len(stored.Data.Experiences))
	}
}

func TestProjectCRUD(t *testing.T) {
	s := seededStore(t)

	index, err := s.AddProject(Project{Name: "New Tool", Tech: "Go", Bullets: []string{"n1", "n2"}})
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if index != 1 {
		t.Fatalf("index = %d, want 1", index)
	}

	if err := s.UpdateProject(index, Project{Name: "Renamed", Tech: "Rust"}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	p, err := s.GetProject(index)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if p.Name != "Renamed" || p.Tech != "Rust" {
		t.Fatalf("project = %+v, want Renamed/Rust", p)
	}

	deleted, err := s.DeleteProject(index)
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if deleted.Name != "Renamed" {
		t.Fatalf("deleted = %+v, want Renamed", deleted)
	}
	if _, err := s.GetProject(index); err == nil {
		t.Fatal("GetProject after delete: expected error")
	}
}

func TestEducationCRUD(t *testing.T) {
	s := seededStore(t)

	index, err := s.AddEducation(Education{Institution: "Grad School", Degree: "M.S."})
	if err != nil {
		t.Fatalf("AddEducation: %v", err)
	}
	if index != 1 {
		t.Fatalf("index = %d, want 1", index)
	}

	if err := s.UpdateEducation(index, Education{Institution: "Grad School", Degree: "M.S. CS"}); err != nil {
		t.Fatalf("UpdateEducation: %v", err)
	}
	e, err := s.GetEducation(index)
	if err != nil {
		t.Fatalf("GetEducation: %v", err)
	}
	if e.Degree != "M.S. CS" {
		t.Fatalf("degree = %q, want M.S. CS", e.Degree)
	}

	deleted, err := s.DeleteEducation(index)
	if err != nil {
		t.Fatalf("DeleteEducation: %v", err)
	}
	if deleted.Degree != "M.S. CS" {
		t.Fatalf("deleted = %+v, want M.S. CS", deleted)
	}
	if _, err := s.GetEducation(index); err == nil {
		t.Fatal("GetEducation after delete: expected error")
	}
}

func TestSkillCRUD(t *testing.T) {
	s := seededStore(t)

	index, err := s.AddSkill(SkillGroup{Category: "Cloud", Values: "AWS, GCP"})
	if err != nil {
		t.Fatalf("AddSkill: %v", err)
	}
	if index != 1 {
		t.Fatalf("index = %d, want 1", index)
	}

	// Update replaces the item; partial patches are merged by MergeSkill at the tool layer.
	if err := s.UpdateSkill(index, SkillGroup{Category: "Cloud", Values: "AWS, GCP, Azure"}); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	skill, err := s.GetSkill(index)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if skill.Category != "Cloud" || skill.Values != "AWS, GCP, Azure" {
		t.Fatalf("skill = %+v, want Cloud/AWS, GCP, Azure", skill)
	}

	deleted, err := s.DeleteSkill(index)
	if err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if deleted.Category != "Cloud" {
		t.Fatalf("deleted = %+v, want Cloud", deleted)
	}
	if _, err := s.GetSkill(index); err == nil {
		t.Fatal("GetSkill after delete: expected error")
	}
}

func TestBulletCRUD(t *testing.T) {
	s := seededStore(t)

	index, err := s.AddBullet(ParentExperience, 0, "a3")
	if err != nil {
		t.Fatalf("AddBullet: %v", err)
	}
	if index != 2 {
		t.Fatalf("index = %d, want 2", index)
	}

	text, err := s.GetBullet(ParentExperience, 0, index)
	if err != nil {
		t.Fatalf("GetBullet: %v", err)
	}
	if text != "a3" {
		t.Fatalf("bullet = %q, want a3", text)
	}

	if err := s.UpdateBullet(ParentExperience, 0, index, "a3-updated"); err != nil {
		t.Fatalf("UpdateBullet: %v", err)
	}
	text, err = s.GetBullet(ParentExperience, 0, index)
	if err != nil {
		t.Fatalf("GetBullet after update: %v", err)
	}
	if text != "a3-updated" {
		t.Fatalf("bullet = %q, want a3-updated", text)
	}

	deleted, err := s.DeleteBullet(ParentExperience, 0, index)
	if err != nil {
		t.Fatalf("DeleteBullet: %v", err)
	}
	if deleted != "a3-updated" {
		t.Fatalf("deleted = %q, want a3-updated", deleted)
	}
	if _, err := s.GetBullet(ParentExperience, 0, index); err == nil {
		t.Fatal("GetBullet after delete: expected error")
	}

	// Project bullets use the same path with a different parent type.
	idx, err := s.AddBullet(ParentProject, 0, "p2")
	if err != nil {
		t.Fatalf("AddBullet project: %v", err)
	}
	if idx != 1 {
		t.Fatalf("project bullet index = %d, want 1", idx)
	}
}

func TestBulletInvalidRefs(t *testing.T) {
	s := seededStore(t)

	if _, err := s.AddBullet("education", 0, "x"); err == nil || !strings.Contains(err.Error(), "parentType") {
		t.Fatalf("bad parentType error = %v, want parentType hint", err)
	}
	if _, err := s.AddBullet(ParentExperience, 99, "x"); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("bad parentIndex error = %v, want out of range", err)
	}
	if _, err := s.AddBullet(ParentExperience, 0, "   "); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("empty bullet error = %v, want required", err)
	}
	if _, err := s.GetBullet(ParentExperience, 0, 99); err == nil || !strings.Contains(err.Error(), "bullet index") {
		t.Fatalf("bad bulletIndex error = %v, want bullet index hint", err)
	}
	if err := s.UpdateBullet(ParentProject, 99, 0, "x"); err == nil {
		t.Fatal("UpdateBullet with bad parent: expected error")
	}
	if _, err := s.DeleteBullet(ParentExperience, 0, -1); err == nil {
		t.Fatal("DeleteBullet with negative index: expected error")
	}
}

func TestInvalidIndices(t *testing.T) {
	s := seededStore(t)

	_, err := s.GetExperience(-1)
	assertOutOfRange(t, "GetExperience(-1)", err)
	_, err = s.GetExperience(99)
	assertOutOfRange(t, "GetExperience(99)", err)
	assertOutOfRange(t, "UpdateExperience(99)", s.UpdateExperience(99, Experience{}))
	_, err = s.DeleteExperience(99)
	assertOutOfRange(t, "DeleteExperience(99)", err)
	_, err = s.GetProject(5)
	assertOutOfRange(t, "GetProject(5)", err)
	assertOutOfRange(t, "UpdateProject(-1)", s.UpdateProject(-1, Project{}))
	_, err = s.GetEducation(2)
	assertOutOfRange(t, "GetEducation(2)", err)
	_, err = s.DeleteEducation(2)
	assertOutOfRange(t, "DeleteEducation(2)", err)
	_, err = s.GetSkill(9)
	assertOutOfRange(t, "GetSkill(9)", err)
	_, err = s.DeleteSkill(9)
	assertOutOfRange(t, "DeleteSkill(9)", err)
}

func assertOutOfRange(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error", name)
		return
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("%s error = %v, want out of range", name, err)
	}
}

func TestCRUDOnEmptyResume(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(ResumeData{Name: "Empty"}); err != nil {
		t.Fatalf("save empty resume: %v", err)
	}

	if _, err := s.GetExperience(0); err == nil {
		t.Fatal("GetExperience on empty resume: expected error")
	}
	if _, err := s.AddBullet(ParentExperience, 0, "x"); err == nil {
		t.Fatal("AddBullet with no parents: expected error")
	}
	if _, err := s.AddSkill(SkillGroup{Category: "Languages", Values: "Go"}); err != nil {
		t.Fatalf("AddSkill on empty resume: %v", err)
	}
	if _, err := s.GetSkill(0); err != nil {
		t.Fatalf("GetSkill after add: %v", err)
	}
	deleted, err := s.DeleteSkill(0)
	if err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if deleted.Category != "Languages" {
		t.Fatalf("deleted = %+v, want Languages", deleted)
	}
	stored, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(stored.Data.Skills) != 0 {
		t.Fatalf("skills = %d, want 0", len(stored.Data.Skills))
	}
}

func TestCRUDPreservesInitializedAt(t *testing.T) {
	s := seededStore(t)
	before, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := s.AddExperience(Experience{Company: "X"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !after.InitializedAt.Equal(before.InitializedAt) {
		t.Fatalf("InitializedAt = %v, want %v (CRUD must not reset it)", after.InitializedAt, before.InitializedAt)
	}
}

func TestMergeHelpers(t *testing.T) {
	exp := MergeExperience(
		Experience{Company: "Acme", Role: "Engineer", Bullets: []string{"b1", "b2"}},
		Experience{Role: "Senior"},
	)
	if exp.Company != "Acme" || exp.Role != "Senior" || len(exp.Bullets) != 2 {
		t.Fatalf("MergeExperience = %+v, want Acme/Senior with 2 bullets", exp)
	}

	exp = MergeExperience(exp, Experience{Bullets: []string{"only"}})
	if len(exp.Bullets) != 1 || exp.Bullets[0] != "only" {
		t.Fatalf("MergeExperience bullets = %v, want [only]", exp.Bullets)
	}

	proj := MergeProject(Project{Name: "Tool", Tech: "Go", Bullets: []string{"p1"}}, Project{Tech: "Rust"})
	if proj.Name != "Tool" || proj.Tech != "Rust" || len(proj.Bullets) != 1 {
		t.Fatalf("MergeProject = %+v, want Tool/Rust with 1 bullet", proj)
	}

	edu := MergeEducation(Education{Institution: "U", Degree: "B.S.", Location: "NYC"}, Education{Degree: "M.S."})
	if edu.Institution != "U" || edu.Degree != "M.S." || edu.Location != "NYC" {
		t.Fatalf("MergeEducation = %+v, want U/M.S./NYC", edu)
	}

	skill := MergeSkill(SkillGroup{Category: "Languages", Values: "Go"}, SkillGroup{Values: "Go, Rust"})
	if skill.Category != "Languages" || skill.Values != "Go, Rust" {
		t.Fatalf("MergeSkill = %+v, want Languages/Go, Rust", skill)
	}
}

func TestValidItemTypes(t *testing.T) {
	for _, typ := range []string{ItemExperience, ItemProject, ItemEducation, ItemSkill, ItemBullet} {
		if !ValidItemType(typ) {
			t.Errorf("ValidItemType(%q) = false, want true", typ)
		}
	}
	if ValidItemType("bogus") {
		t.Error("ValidItemType(bogus) = true, want false")
	}
	if !ValidParentType(ParentExperience) || !ValidParentType(ParentProject) {
		t.Error("ValidParentType rejected a valid parent type")
	}
	if ValidParentType(ItemBullet) {
		t.Error("ValidParentType(bullet) = true, want false")
	}
}
