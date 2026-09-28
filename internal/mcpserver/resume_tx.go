package mcpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// itemTx stages resume item mutations in memory and commits them together:
// pending embeddings are computed in one pass, then the resume and vector store
// are written once each. An uncommitted tx leaves disk untouched, so single
// operations and batches apply fully or not at all.
type itemTx struct {
	deps   deps
	data   resume.ResumeData
	chunks []vectorstore.Chunk
	initAt time.Time
}

func newItemTx(d deps) (*itemTx, error) {
	stored, err := d.ResumeStore.Load()
	if err != nil {
		return nil, err
	}
	return &itemTx{
		deps:   d,
		data:   stored.Data,
		chunks: d.VectorStore.Snapshot(),
		initAt: stored.InitializedAt,
	}, nil
}

// newInitTx stages a full overwrite, so all content is re-embedded on commit.
func newInitTx(d deps, data resume.ResumeData) *itemTx {
	return &itemTx{
		deps:   d,
		data:   data,
		chunks: vectorstore.BuildChunks(data),
		initAt: time.Now(),
	}
}

func (tx *itemTx) apply(args ResumeItemsInput) (ItemEntry, error) {
	switch args.Operation {
	case "add":
		return tx.add(args)
	case "update":
		return tx.update(args)
	case "delete":
		return tx.delete(args)
	default:
		return ItemEntry{}, fmt.Errorf("batch supports 'add', 'update', and 'delete' requests (got %q)", args.Operation)
	}
}

// stage merges desired chunks into the staged set. Unchanged text keeps its
// embedding; new or changed text is left unembedded for commit.
func (tx *itemTx) stage(desired []vectorstore.Chunk, staleIDs []string) {
	tx.chunks = vectorstore.SyncChunkSlice(tx.chunks, desired, nil, staleIDs)
}

// add/update/delete stage one request onto the tx.

func (tx *itemTx) add(args ResumeItemsInput) (ItemEntry, error) {
	if args.Item == nil {
		return ItemEntry{}, fmt.Errorf("item is required for add")
	}

	switch args.Type {
	case resume.ItemExperience:
		item := args.Item.toExperience()
		index := resume.ApplyAddExperience(&tx.data, item)
		tx.stage(experienceChunks(item, index), nil)
		return experienceEntry(index, item), nil

	case resume.ItemProject:
		item := args.Item.toProject()
		index := resume.ApplyAddProject(&tx.data, item)
		tx.stage(projectChunks(item, index), nil)
		return projectEntry(index, item), nil

	case resume.ItemEducation:
		item := args.Item.toEducation()
		index := resume.ApplyAddEducation(&tx.data, item)
		tx.stage([]vectorstore.Chunk{vectorstore.EducationChunk(item, index)}, nil)
		return educationEntry(index, item), nil

	case resume.ItemSkill:
		item := args.Item.toSkill()
		index := resume.ApplyAddSkill(&tx.data, item)
		tx.stage([]vectorstore.Chunk{vectorstore.SkillChunk(item, index)}, nil)
		return skillEntry(index, item), nil

	case resume.ItemBullet:
		parentType, parentIndex, err := bulletParent(args)
		if err != nil {
			return ItemEntry{}, err
		}
		text := args.Item.Text
		if strings.TrimSpace(text) == "" {
			return ItemEntry{}, fmt.Errorf("item.text is required when adding a bullet")
		}
		bullets, err := tx.data.Bullets(parentType, parentIndex)
		if err != nil {
			return ItemEntry{}, err
		}
		index := len(bullets)
		chunk, err := bulletChunk(tx.data, parentType, parentIndex, index, text)
		if err != nil {
			return ItemEntry{}, err
		}
		if _, err := resume.ApplyAddBullet(&tx.data, parentType, parentIndex, text); err != nil {
			return ItemEntry{}, err
		}
		tx.stage([]vectorstore.Chunk{chunk}, nil)
		return bulletEntry(parentType, parentIndex, index, text), nil

	default:
		return ItemEntry{}, invalidTypeError(args.Type)
	}
}

func (tx *itemTx) update(args ResumeItemsInput) (ItemEntry, error) {
	if args.Item == nil {
		return ItemEntry{}, fmt.Errorf("item is required for update")
	}

	switch args.Type {
	case resume.ItemExperience:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		old, err := tx.data.ExperienceAt(index)
		if err != nil {
			return ItemEntry{}, err
		}
		merged := resume.MergeExperience(old, args.Item.toExperience())
		if err := resume.ApplyUpdateExperience(&tx.data, index, merged); err != nil {
			return ItemEntry{}, err
		}
		stale := staleBulletIDs(vectorstore.ExperienceChunkID, index, len(merged.Bullets), len(old.Bullets))
		tx.stage(experienceChunks(merged, index), stale)
		return experienceEntry(index, merged), nil

	case resume.ItemProject:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		old, err := tx.data.ProjectAt(index)
		if err != nil {
			return ItemEntry{}, err
		}
		merged := resume.MergeProject(old, args.Item.toProject())
		if err := resume.ApplyUpdateProject(&tx.data, index, merged); err != nil {
			return ItemEntry{}, err
		}
		stale := staleBulletIDs(vectorstore.ProjectChunkID, index, len(merged.Bullets), len(old.Bullets))
		tx.stage(projectChunks(merged, index), stale)
		return projectEntry(index, merged), nil

	case resume.ItemEducation:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		old, err := tx.data.EducationAt(index)
		if err != nil {
			return ItemEntry{}, err
		}
		merged := resume.MergeEducation(old, args.Item.toEducation())
		if err := resume.ApplyUpdateEducation(&tx.data, index, merged); err != nil {
			return ItemEntry{}, err
		}
		tx.stage([]vectorstore.Chunk{vectorstore.EducationChunk(merged, index)}, nil)
		return educationEntry(index, merged), nil

	case resume.ItemSkill:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		old, err := tx.data.SkillAt(index)
		if err != nil {
			return ItemEntry{}, err
		}
		merged := resume.MergeSkill(old, args.Item.toSkill())
		if err := resume.ApplyUpdateSkill(&tx.data, index, merged); err != nil {
			return ItemEntry{}, err
		}
		tx.stage([]vectorstore.Chunk{vectorstore.SkillChunk(merged, index)}, nil)
		return skillEntry(index, merged), nil

	case resume.ItemBullet:
		parentType, parentIndex, err := bulletParent(args)
		if err != nil {
			return ItemEntry{}, err
		}
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		bullets, err := tx.data.Bullets(parentType, parentIndex)
		if err != nil {
			return ItemEntry{}, err
		}
		if index < 0 || index >= len(bullets) {
			return ItemEntry{}, fmt.Errorf("bullet index %d out of range (have %d)", index, len(bullets))
		}
		text := args.Item.Text
		if strings.TrimSpace(text) == "" {
			return ItemEntry{}, fmt.Errorf("item.text is required when updating a bullet")
		}
		chunk, err := bulletChunk(tx.data, parentType, parentIndex, index, text)
		if err != nil {
			return ItemEntry{}, err
		}
		if err := resume.ApplyUpdateBullet(&tx.data, parentType, parentIndex, index, text); err != nil {
			return ItemEntry{}, err
		}
		tx.stage([]vectorstore.Chunk{chunk}, nil)
		return bulletEntry(parentType, parentIndex, index, text), nil

	default:
		return ItemEntry{}, invalidTypeError(args.Type)
	}
}

func (tx *itemTx) delete(args ResumeItemsInput) (ItemEntry, error) {
	switch args.Type {
	case resume.ItemExperience:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		deleted, err := resume.ApplyDeleteExperience(&tx.data, index)
		if err != nil {
			return ItemEntry{}, err
		}
		tx.chunks = vectorstore.EraseExperience(tx.chunks, index)
		return experienceEntry(index, deleted), nil

	case resume.ItemProject:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		deleted, err := resume.ApplyDeleteProject(&tx.data, index)
		if err != nil {
			return ItemEntry{}, err
		}
		tx.chunks = vectorstore.EraseProject(tx.chunks, index)
		return projectEntry(index, deleted), nil

	case resume.ItemEducation:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		deleted, err := resume.ApplyDeleteEducation(&tx.data, index)
		if err != nil {
			return ItemEntry{}, err
		}
		tx.chunks = vectorstore.EraseEducation(tx.chunks, index)
		return educationEntry(index, deleted), nil

	case resume.ItemSkill:
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		deleted, err := resume.ApplyDeleteSkill(&tx.data, index)
		if err != nil {
			return ItemEntry{}, err
		}
		tx.chunks = vectorstore.EraseSkill(tx.chunks, index)
		return skillEntry(index, deleted), nil

	case resume.ItemBullet:
		parentType, parentIndex, err := bulletParent(args)
		if err != nil {
			return ItemEntry{}, err
		}
		index, err := requireID(args)
		if err != nil {
			return ItemEntry{}, err
		}
		deleted, err := resume.ApplyDeleteBullet(&tx.data, parentType, parentIndex, index)
		if err != nil {
			return ItemEntry{}, err
		}
		tx.chunks = vectorstore.EraseBullet(tx.chunks, parentType, parentIndex, index)
		return bulletEntry(parentType, parentIndex, index, deleted), nil

	default:
		return ItemEntry{}, invalidTypeError(args.Type)
	}
}

// commit embeds the staged chunks that lack an embedding, then writes the
// resume and vector store. Embedding failures abort before anything is
// persisted.
func (tx *itemTx) commit() error {
	var pending []int
	var texts []string
	for i := range tx.chunks {
		if tx.chunks[i].Embedding == nil {
			pending = append(pending, i)
			texts = append(texts, tx.chunks[i].Text)
		}
	}
	if len(texts) > 0 {
		cfg, err := tx.deps.ConfigStore.Load()
		if err != nil {
			return fmt.Errorf("embedding config required: %w", err)
		}
		embeddings, err := vectorstore.NewEmbedClient(*cfg).EmbedBatch(texts)
		if err != nil {
			return fmt.Errorf("embed items: %w", err)
		}
		for k, i := range pending {
			tx.chunks[i].Embedding = embeddings[k]
		}
	}

	if err := tx.deps.ResumeStore.SaveStored(resume.StoredResume{Data: tx.data, InitializedAt: tx.initAt}); err != nil {
		return err
	}
	tx.deps.VectorStore.SetChunks(tx.chunks)
	if err := tx.deps.VectorStore.Save(); err != nil {
		return fmt.Errorf("save vector store: %w", err)
	}
	return nil
}

func (tx *itemTx) stats() resume.Stats {
	return resume.ComputeStats(tx.data)
}

func (tx *itemTx) chunkCount() int {
	return len(tx.chunks)
}
