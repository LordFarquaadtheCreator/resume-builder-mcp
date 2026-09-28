package vectorstore

import (
	"fmt"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

func (s *Store) GetChunk(id string) (Chunk, bool) {
	return getChunk(s.chunks, id)
}

// AddChunk fails if a chunk with the same ID exists.
func (s *Store) AddChunk(c Chunk) error {
	if _, exists := s.GetChunk(c.ID); exists {
		return fmt.Errorf("chunk %q already exists", c.ID)
	}
	s.chunks = append(s.chunks, c)
	return nil
}

// UpdateChunk keeps the given ID, ignoring c.ID, and fails on an unknown ID.
func (s *Store) UpdateChunk(id string, c Chunk) error {
	for i := range s.chunks {
		if s.chunks[i].ID == id {
			c.ID = id
			s.chunks[i] = c
			return nil
		}
	}
	return fmt.Errorf("chunk %q not found", id)
}

// DeleteChunk fails if no chunk has the ID.
func (s *Store) DeleteChunk(id string) error {
	for i := range s.chunks {
		if s.chunks[i].ID == id {
			s.chunks = append(s.chunks[:i], s.chunks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("chunk %q not found", id)
}

// NeedsReembed reports whether a chunk must be embedded before it can be
// applied: true when no chunk with the ID exists or its text differs.
func (s *Store) NeedsReembed(id, text string) bool {
	c, ok := getChunk(s.chunks, id)
	return !ok || c.Text != text
}

// Snapshot returns a copy of all chunks for staging changes that are
// committed later with SetChunks.
func (s *Store) Snapshot() []Chunk {
	return append([]Chunk(nil), s.chunks...)
}

// SetChunks replaces all chunks and takes ownership of the slice.
func (s *Store) SetChunks(chunks []Chunk) {
	s.chunks = chunks
}

func (s *Store) SyncChunks(desired []Chunk, embeddings map[string][]float64, staleIDs []string) {
	s.chunks = SyncChunkSlice(s.chunks, desired, embeddings, staleIDs)
}

// Delete* remove one item's chunks and renumber later items down by one so IDs
// keep matching resume positions.
func (s *Store) DeleteExperience(index int) {
	s.chunks = EraseExperience(s.chunks, index)
}

func (s *Store) DeleteProject(index int) {
	s.chunks = EraseProject(s.chunks, index)
}

func (s *Store) DeleteSkill(index int) {
	s.chunks = EraseSkill(s.chunks, index)
}

func (s *Store) DeleteEducation(index int) {
	s.chunks = EraseEducation(s.chunks, index)
}

func (s *Store) DeleteBulletFromParent(parentType string, parentIndex, bulletIndex int) {
	s.chunks = EraseBullet(s.chunks, parentType, parentIndex, bulletIndex)
}

// --- pure slice operations, used for staged updates ---

// SyncChunkSlice returns chunks matching desired: desired chunks are inserted
// or replaced, staleIDs are dropped, and unchanged text keeps its embedding. A
// changed chunk with no supplied embedding is left nil and must be embedded by
// the caller before use.
func SyncChunkSlice(chunks []Chunk, desired []Chunk, embeddings map[string][]float64, staleIDs []string) []Chunk {
	out := append([]Chunk(nil), chunks...)
	for _, c := range desired {
		existing, found := getChunk(out, c.ID)
		if found && existing.Text == c.Text {
			c.Embedding = existing.Embedding
		} else if emb, ok := embeddings[c.ID]; ok {
			c.Embedding = emb
		} else {
			c.Embedding = nil
		}
		if found {
			out = replaceChunk(out, c)
		} else {
			out = append(out, c)
		}
	}
	for _, id := range staleIDs {
		out = removeChunk(out, id)
	}
	return out
}

// Erase* mirror the Store.Delete* methods on a chunk slice.
func EraseExperience(chunks []Chunk, index int) []Chunk {
	return eraseParent(chunks, experienceCodec, index)
}

func EraseProject(chunks []Chunk, index int) []Chunk {
	return eraseParent(chunks, projectCodec, index)
}

func EraseSkill(chunks []Chunk, index int) []Chunk {
	return eraseParent(chunks, skillCodec, index)
}

func EraseEducation(chunks []Chunk, index int) []Chunk {
	return eraseParent(chunks, educationCodec, index)
}

func EraseBullet(chunks []Chunk, parentType string, parentIndex, bulletIndex int) []Chunk {
	switch parentType {
	case resume.ParentExperience:
		return eraseChild(chunks, experienceCodec, parentIndex, bulletIndex)
	case resume.ParentProject:
		return eraseChild(chunks, projectCodec, parentIndex, bulletIndex)
	default:
		return chunks
	}
}

// --- internals ---

func getChunk(chunks []Chunk, id string) (Chunk, bool) {
	for _, c := range chunks {
		if c.ID == id {
			return c, true
		}
	}
	return Chunk{}, false
}

func replaceChunk(chunks []Chunk, c Chunk) []Chunk {
	for i := range chunks {
		if chunks[i].ID == c.ID {
			chunks[i] = c
			return chunks
		}
	}
	return append(chunks, c)
}

func removeChunk(chunks []Chunk, id string) []Chunk {
	for i := range chunks {
		if chunks[i].ID == id {
			return append(chunks[:i], chunks[i+1:]...)
		}
	}
	return chunks
}

func eraseParent(chunks []Chunk, c chunkCodec, index int) []Chunk {
	out := make([]Chunk, 0, len(chunks))
	for _, ch := range chunks {
		parent, child, ok := c.parse(ch.ID)
		if !ok {
			out = append(out, ch)
			continue
		}
		if parent == index {
			continue
		}
		if parent > index {
			ch.ID = c.rebuild(parent-1, child)
		}
		out = append(out, ch)
	}
	return out
}

func eraseChild(chunks []Chunk, c chunkCodec, parent, child int) []Chunk {
	out := make([]Chunk, 0, len(chunks))
	for _, ch := range chunks {
		p, k, ok := c.parse(ch.ID)
		if !ok {
			out = append(out, ch)
			continue
		}
		if p == parent && k == child {
			continue
		}
		if p == parent && k > child {
			ch.ID = c.rebuild(p, k-1)
		}
		out = append(out, ch)
	}
	return out
}
