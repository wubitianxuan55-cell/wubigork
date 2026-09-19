package dag

// 流水线模板库（6.3 余项，设计 docs/gaea-office-dag-63-design-2026-09.md §6）：
// 「月度报告」这类周期性流水线存成模板，一键重建为新的草稿 run。真相=
// <cwd>/.gaea/work/dag/templates/<id>.json（run 档同域子目录——Store.List 只读
// 顶层 *.json，子目录天然不混；删档即弃不进用户库）。模板只取图形状
// （goal+节点指令+依赖），状态/产物/运行痕迹一律剥掉。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// NameMaxRune 模板名上限（rune 计，中文友好）；超限报错不静默截断。
const NameMaxRune = 40

// Template 一条可复用的流水线形状。
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"` // 人读名（如「月度经营报告」）
	Goal        string `json:"goal"`
	Nodes       []Node `json:"nodes"` // 只含 id/title/prompt/dependsOn（Status=pending）
	CreatedAt   string `json:"createdAt"`
	SourceRunID string `json:"sourceRunId,omitempty"` // 由哪条 run 存来（溯源）
}

// tplSeq 进程内原子序号：模板 id 与 run id 同秒不撞名（NewID 同款）。
var tplSeq atomic.Uint64

// NewTemplateID 模板 id：时间戳+进程内序号 slug。
func NewTemplateID() string {
	return fmt.Sprintf("dagtpl_%s_%d", time.Now().Format("20060102_150405"), tplSeq.Add(1))
}

// normalizeName 模板名收口：trim+空回退 goal 截断+限长校验。空名不报错——
// 「存为模板」的最顺手形态就是不带名字（用 goal 当名）。
func normalizeName(name, goal string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		goal = strings.TrimSpace(goal)
		if r := []rune(goal); len(r) > 24 {
			goal = string(r[:24])
		}
		name = goal
	}
	if name == "" {
		return "", fmt.Errorf("模板名不能为空（goal 也为空）")
	}
	if len([]rune(name)) > NameMaxRune {
		return "", fmt.Errorf("模板名超长（最多 %d 字）", NameMaxRune)
	}
	return name, nil
}

// FromRun 把既有 run 收成模板：剥掉状态/产物/ref/运行痕迹，节点回 pending。
func FromRun(r Run, name string) (Template, error) {
	nm, err := normalizeName(name, r.Goal)
	if err != nil {
		return Template{}, err
	}
	if err := Validate(r.Goal, r.Nodes); err != nil {
		return Template{}, err
	}
	nodes := make([]Node, 0, len(r.Nodes))
	for _, n := range r.Nodes {
		nodes = append(nodes, Node{
			ID:        n.ID,
			Title:     n.Title,
			Prompt:    n.Prompt,
			DependsOn: n.DependsOn,
			Risk:      n.Risk,
			Status:    StatusPending,
		})
	}
	return Template{
		ID:          NewTemplateID(),
		Name:        nm,
		Goal:        strings.TrimSpace(r.Goal),
		Nodes:       nodes,
		CreatedAt:   time.Now().Format(time.RFC3339),
		SourceRunID: r.ID,
	}, nil
}

// Instantiate 模板→新草稿 run：全新 run id、全节点 pending、无产物无运行痕迹。
// 不自动起跑——起跑仍是人拍板（与整链首跑同闸）。
func Instantiate(t Template) Run {
	nodes := make([]Node, 0, len(t.Nodes))
	for _, n := range t.Nodes {
		nodes = append(nodes, Node{
			ID:        n.ID,
			Title:     n.Title,
			Prompt:    n.Prompt,
			DependsOn: n.DependsOn,
			Risk:      n.Risk,
			Status:    StatusPending,
		})
	}
	return Run{
		ID:        NewID(),
		Goal:      t.Goal,
		CreatedAt: time.Now().Format(time.RFC3339),
		Nodes:     nodes,
	}
}

// TemplateStore 模板档文件存储（无状态；并发读改写由调用方串行化）。
type TemplateStore struct{ dir string }

func NewTemplateStore(dir string) *TemplateStore { return &TemplateStore{dir: dir} }

func (s *TemplateStore) path(id string) string { return filepath.Join(s.dir, id+".json") }

// Save 原子落盘（临时文件+改名）；落盘前全量校验（goal/节点/依赖/成环）——
// 模板是「可重建」的承诺，坏形状不能入库。
func (s *TemplateStore) Save(t Template) error {
	if err := safeID(t.ID); err != nil {
		return err
	}
	if err := Validate(t.Goal, t.Nodes); err != nil {
		return err
	}
	nm, err := normalizeName(t.Name, t.Goal)
	if err != nil {
		return err
	}
	t.Name = nm
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("创建模板目录: %w", err)
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path(t.ID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return fileutil.RenameWithRetry(tmp, s.path(t.ID))
}

// Get 读单条；不存在报错（fail-closed）。
func (s *TemplateStore) Get(id string) (Template, error) {
	if err := safeID(id); err != nil {
		return Template{}, err
	}
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		return Template{}, fmt.Errorf("读取模板 %s: %w", id, err)
	}
	var t Template
	if err := json.Unmarshal(b, &t); err != nil {
		return Template{}, fmt.Errorf("解析模板 %s: %w", id, err)
	}
	return t, nil
}

// List 全部模板（创建时间倒序）；损坏文件跳过（只读兼容，镜像 Store.List）。
func (s *TemplateStore) List() ([]Template, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		// 目录不存在=空集语义：nil slice 经 JSON 序列化成 null，会把前端
		// 非空断言的消费方（DagPanel tpls.length）炸掉（v4.234 真机走查实锤）。
		return []Template{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Template
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var t Template
		if json.Unmarshal(b, &t) == nil && t.ID != "" {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// Delete 删档即弃；不存在报错（fail-closed，不静默成功）。
func (s *TemplateStore) Delete(id string) error {
	if err := safeID(id); err != nil {
		return err
	}
	if err := os.Remove(s.path(id)); err != nil {
		return fmt.Errorf("删除模板 %s: %w", id, err)
	}
	return nil
}
