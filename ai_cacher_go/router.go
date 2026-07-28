// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

package ai_cacher_go

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CachedEntry struct {
	Vector  []float64
	Prompt  string
	Payload map[string]any
}

type CacheDocument struct {
	Prompt     string         `bson:"prompt"`
	PromptHash string         `bson:"promptHash"`
	Vector     []float64      `bson:"vector"`
	Payload    map[string]any `bson:"payload"`
	UpdatedAt  time.Time      `bson:"updatedAt"`
}

type CacheStore interface {
	FindByPromptHash(ctx context.Context, promptHash string) (*CacheDocument, error)
	FindAll(ctx context.Context) ([]CacheDocument, error)
	Upsert(ctx context.Context, doc CacheDocument) error
}

type MongoCacheStore struct {
	collection *mongo.Collection
}

func NewMongoCacheStore(ctx context.Context, uri, dbName, collectionName string) (*MongoCacheStore, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	collection := client.Database(dbName).Collection(collectionName)
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "promptHash", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return &MongoCacheStore{collection: collection}, nil
}

func (m *MongoCacheStore) FindByPromptHash(ctx context.Context, promptHash string) (*CacheDocument, error) {
	var doc CacheDocument
	if err := m.collection.FindOne(ctx, bson.M{"promptHash": promptHash}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}

func (m *MongoCacheStore) FindAll(ctx context.Context) ([]CacheDocument, error) {
	cursor, err := m.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []CacheDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (m *MongoCacheStore) Upsert(ctx context.Context, doc CacheDocument) error {
	_, err := m.collection.UpdateOne(
		ctx,
		bson.M{"promptHash": doc.PromptHash},
		bson.M{"$set": doc},
		options.Update().SetUpsert(true),
	)
	return err
}

type SemanticCacheRouter struct {
	Threshold float64
	Cache     map[int]CachedEntry
	Counter   int
	Store     CacheStore
}

func NewSemanticCacheRouter(threshold float64, store CacheStore) *SemanticCacheRouter {
	if threshold <= 0 {
		threshold = 0.95
	}
	return &SemanticCacheRouter{
		Threshold: threshold,
		Cache:     make(map[int]CachedEntry),
		Store:     store,
	}
}

func hashPrompt(prompt string) string {
	h := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(h[:])
}

func (r *SemanticCacheRouter) getEmbedding(text string) []float64 {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return make([]float64, 128)
	}

	tokens := strings.Fields(text)
	freq := make(map[string]float64)
	for _, token := range tokens {
		freq[token] += 1
	}

	vector := make([]float64, 128)
	keys := make([]string, 0, len(freq))
	for token := range freq {
		keys = append(keys, token)
	}
	sort.Strings(keys)
	for i, token := range keys {
		if i >= 128 {
			break
		}
		vector[i] = freq[token]
	}

	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vector
	}
	for i := range vector {
		vector[i] /= norm
	}
	return vector
}

func (r *SemanticCacheRouter) CheckCache(ctx context.Context, prompt string) (map[string]any, error) {
	if r.Store != nil {
		promptHash := hashPrompt(prompt)
		exactDoc, err := r.Store.FindByPromptHash(ctx, promptHash)
		if err != nil {
			return nil, err
		}
		if exactDoc != nil {
			return exactDoc.Payload, nil
		}

		docs, err := r.Store.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		if len(docs) == 0 {
			return nil, nil
		}

		newVec := r.getEmbedding(prompt)
		bestScore := 0.0
		var bestPayload map[string]any
		for _, doc := range docs {
			dot := 0.0
			for i, value := range newVec {
				dot += value * doc.Vector[i]
			}
			if dot > bestScore {
				bestScore = dot
				bestPayload = doc.Payload
			}
		}
		if bestScore >= r.Threshold {
			return bestPayload, nil
		}
		return nil, nil
	}

	if len(r.Cache) == 0 {
		return nil, nil
	}

	newVec := r.getEmbedding(prompt)
	bestScore := 0.0
	var bestPayload map[string]any
	for _, entry := range r.Cache {
		dot := 0.0
		for i, value := range newVec {
			dot += value * entry.Vector[i]
		}
		if dot > bestScore {
			bestScore = dot
			bestPayload = entry.Payload
		}
	}

	if bestScore >= r.Threshold {
		return bestPayload, nil
	}
	return nil, nil
}

func (r *SemanticCacheRouter) UpdateCache(ctx context.Context, prompt string, response map[string]any) error {
	vec := r.getEmbedding(prompt)
	if r.Store != nil {
		promptHash := hashPrompt(prompt)
		return r.Store.Upsert(ctx, CacheDocument{
			Prompt:     prompt,
			PromptHash: promptHash,
			Vector:     vec,
			Payload:    response,
			UpdatedAt:  time.Now(),
		})
	}

	r.Cache[r.Counter] = CachedEntry{Vector: vec, Prompt: prompt, Payload: response}
	r.Counter++
	return nil
}
