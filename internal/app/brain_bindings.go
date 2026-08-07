package app

import (
	"encoding/json"
	"strings"

	gaeacfg "github.com/gaea/gaea/internal/gaea/config"
	gaeadb "github.com/gaea/gaea/internal/gaea/db"
	gaekb "github.com/gaea/gaea/internal/gaea/knowledge"
	gaemem "github.com/gaea/gaea/internal/gaea/memory"
)

// initBrain 装配三脑统一访问层（零迁移：适配器直接包现有存储）。
func (a *App) initBrain() {
	gdb := gaeadb.GetDatabase(gaeacfg.MemoryUserDir())
	kbStore, _ := gaekb.Global().Store()
	a.brain = &BrainStore{
		main:  &mainBrain{profile: gaemem.NewProfileStore(gdb), kb: kbStore},
		left:  &leftBrain{src: &proposalLeftSource{svc: a.proposalSvc}},
		right: &rightBrain{dataRoot: a.whisperDataRoot},
		links: NewLinkStore(gdb),
	}
}

// BrainWrite 写入指定脑的事实（主脑/右脑可写；左脑为只读业务域）。
func (a *App) BrainWrite(brain, entity, attribute, value string) error {
	if a.brain == nil {
		return nil
	}
	return a.brain.Write(brain, entity, attribute, value)
}

// BrainSearch 跨脑检索（brains 逗号分隔，空 = 三脑全搜）。
func (a *App) BrainSearch(query, brains string) (string, error) {
	var names []string
	if brains != "" {
		for _, n := range strings.Split(brains, ",") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
	}
	var hits []Hit
	if a.brain != nil {
		var err error
		hits, err = a.brain.Search(query, names...)
		if err != nil {
			return "", err
		}
	}
	b, _ := json.Marshal(hits)
	return string(b), nil
}

// BrainCrossRefs 返回实体的跨脑关联。
func (a *App) BrainCrossRefs(entity string) (string, error) {
	var refs []Ref
	if a.brain != nil {
		var err error
		refs, err = a.brain.CrossRefs(entity)
		if err != nil {
			return "", err
		}
	}
	b, _ := json.Marshal(refs)
	return string(b), nil
}
