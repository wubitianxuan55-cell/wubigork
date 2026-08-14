// Package repos — fact_embeddings 事实向量嵌入仓库
// 100% 对齐 ackem src/main/db/repos/factEmbeddingsRepo.ts
package repos

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// ComputeCorpusHash 计算事实摘要的 SHA256 哈希（用于增量更新判断）
func ComputeCorpusHash(facts []whisper.MemoryFact) string {
	h := sha256.New()
	// 按 ID 排序确保确定性
	ids := make([]string, len(facts))
	for i, f := range facts {
		ids[i] = f.ID
	}
	sort.Strings(ids)

	lookup := make(map[string]whisper.MemoryFact, len(facts))
	for _, f := range facts {
		lookup[f.ID] = f
	}

	for _, id := range ids {
		f := lookup[id]
		h.Write([]byte(f.ID))
		h.Write([]byte(f.Summary))
		h.Write([]byte(f.Subject))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetStoredCorpusHash 获取已存储的语料指纹
func GetStoredCorpusHash(dataRoot, modelSig string) (string, error) {
	return KVGet(dataRoot, "fact_embeddings_corpus", modelSig)
}

// SetStoredCorpusHash 存储语料指纹
func SetStoredCorpusHash(dataRoot, modelSig, hash string) error {
	return KVSet(dataRoot, "fact_embeddings_corpus", modelSig, hash)
}

// LoadFactEmbeddings 加载指定模型签名的所有向量
func LoadFactEmbeddings(dataRoot, modelSig string) (map[string][]float64, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query(
		"SELECT fact_id, vector, dim FROM fact_embeddings WHERE model_sig = ?",
		modelSig,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]float64)
	for rows.Next() {
		var factID string
		var vector []byte
		var dim int
		if err := rows.Scan(&factID, &vector, &dim); err != nil {
			continue
		}
		vec := bytesToFloat64(vector, dim)
		if vec != nil {
			result[factID] = vec
		}
	}
	return result, nil
}

// UpsertFactEmbeddings 批量 upsert 向量嵌入
func UpsertFactEmbeddings(dataRoot, modelSig string, entries map[string][]float64) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	updatedAt := time.Now().Format(time.RFC3339)

	for factID, vec := range entries {
		vector := float64ToBytes(vec)
		_, err := sqlDB.Exec(
			`INSERT INTO fact_embeddings(fact_id, model_sig, dim, updated_at, vector)
			 VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(fact_id, model_sig) DO UPDATE SET
			   dim = excluded.dim,
			   updated_at = excluded.updated_at,
			   vector = excluded.vector`,
			factID, modelSig, len(vec), updatedAt, vector,
		)
		if err != nil {
			return fmt.Errorf("upsert embedding %s 失败: %w", factID, err)
		}
	}
	return nil
}

// DeleteStaleFactEmbeddings 删除不在活跃事实集中的旧向量
func DeleteStaleFactEmbeddings(dataRoot, modelSig string, activeFactIDs map[string]bool) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query(
		"SELECT fact_id FROM fact_embeddings WHERE model_sig = ?",
		modelSig,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var factID string
		if err := rows.Scan(&factID); err != nil {
			continue
		}
		if !activeFactIDs[factID] {
			toDelete = append(toDelete, factID)
		}
	}

	for _, factID := range toDelete {
		sqlDB.Exec(
			"DELETE FROM fact_embeddings WHERE fact_id = ? AND model_sig = ?",
			factID, modelSig,
		)
	}
	return nil
}

// DeleteFactEmbeddingsForModel 删除整个模型签名的所有向量
func DeleteFactEmbeddingsForModel(dataRoot, modelSig string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	_, err := sqlDB.Exec("DELETE FROM fact_embeddings WHERE model_sig = ?", modelSig)
	return err
}

// ─── float64 ↔ bytes ─────────────────────────────────────────────

func float64ToBytes(vec []float64) []byte {
	buf := make([]byte, len(vec)*8)
	for i, v := range vec {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func bytesToFloat64(data []byte, dim int) []float64 {
	if len(data) < dim*8 {
		return nil
	}
	vec := make([]float64, dim)
	for i := 0; i < dim; i++ {
		vec[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return vec
}
