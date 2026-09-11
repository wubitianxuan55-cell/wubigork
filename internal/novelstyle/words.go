package novelstyle

// 词表数据资产（v4.225 规范知识出内核，用户拍板红线 2026-09-11：领域词表/
// 规则表不写死在 Go 代码里）：默认表从 words.json go:embed（数据与逻辑分离，
// genui handbook 先例）；运行时可被覆盖文件**整体替换**——惯例路径
// .gaea/skills/novel-deslop/words.json（app 层接线，见 app ensureNovelStyleWords），
// 改词表不改代码不发版。引擎（打分/分词/替换）是通用机制，留码。

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// wordTables 是可替换的词表集合。
type wordTables struct {
	Replacements map[string]string `json:"replacements"` // AI 高频词 → 平实替代表（去味第一梯队）
	Blacklist    []string          `json:"blacklist"`    // AI 高频词黑名单（打分规则 9 + 分词词表）
}

//go:embed words.json
var wordsJSON []byte

// words 词表快照（原子换出）。快照持有者用包级变量初始化（先于一切 init 函数）
// ——segment.go 的 vocab 构建在 init 里就要读它，init 函数间的顺序不可依赖。
var words = newWordsSnapshot()

func newWordsSnapshot() *atomic.Pointer[wordTables] {
	p := &atomic.Pointer[wordTables]{}
	p.Store(mustLoadEmbeddedWords())
	return p
}

func mustLoadEmbeddedWords() *wordTables {
	t := &wordTables{}
	if err := json.Unmarshal(wordsJSON, t); err != nil {
		// 内置词表随构建走，损坏属构建事故——fail-fast 暴露，不带病运行。
		panic(fmt.Sprintf("novelstyle 内置词表损坏: %v", err))
	}
	return t
}

func currentWords() *wordTables { return words.Load() }

// LoadWordsFile 用覆盖文件整体替换词表（replacements 与 blacklist 均整表替换，
// 不与默认表合并——语义可预测；想增条目把默认表抄进覆盖文件再改）。分词词表
// 随之重建。文件不存在返回 nil（无覆盖=内置默认）。
func LoadWordsFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	t := &wordTables{}
	if err := json.Unmarshal(b, t); err != nil {
		return fmt.Errorf("词表覆盖文件解析失败 %s: %w", path, err)
	}
	if len(t.Replacements) == 0 && len(t.Blacklist) == 0 {
		return fmt.Errorf("词表覆盖文件无有效内容 %s", path)
	}
	words.Store(t)
	rebuildVocab()
	return nil
}
