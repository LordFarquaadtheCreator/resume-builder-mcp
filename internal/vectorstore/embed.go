package vectorstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbedClient calls an OpenAI-compatible /v1/embeddings endpoint.
type EmbedClient struct {
	cfg    EmbeddingConfig
	client *http.Client
}

func NewEmbedClient(cfg EmbeddingConfig) *EmbedClient {
	return &EmbedClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// Embed returns the embedding vector for the given text.
func (c *EmbedClient) Embed(text string) ([]float64, error) {
	body, err := json.Marshal(embedRequest{Model: c.cfg.Model, Input: text})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	url := c.cfg.BaseURL + "/v1/embeddings"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed request failed: %s: %s", resp.Status, string(raw))
	}

	var eres embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&eres); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(eres.Data) == 0 {
		return nil, fmt.Errorf("embed response: no data")
	}
	return eres.Data[0].Embedding, nil
}

// EmbedBatch embeds multiple texts. Returns embeddings in order.
func (c *EmbedClient) EmbedBatch(texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		emb, err := c.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("embed batch[%d]: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}
