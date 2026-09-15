// Package skillstats 结晶技能调用计数的纯函数内核（7.2-2 判据②：技能被实际
// 调用 ≥5 次且成功率可见）。零 IO 表驱动，先例 routesuggest/taskinbox——落盘/
// 读写由 internal/app 层持有，本包只做累计与视图。
//
// 诚实口径（V1 裁决）：成功率=工具级——read_skill 找到技能并交付正文=ok、
// 找不得=fail；run_skill 管线执行无错=ok、报错=fail。回合级成功率（用户满意
// 与否）留观察池。
package skillstats

import "strings"

// MaxSkills 防膨胀：新名拒收上限（既有名更新不受限）。名字来源是真实技能
// 目录，正常远达不到；上限只防名字漂移长期堆积。
const MaxSkills = 500

// Stat 单技能累计。
type Stat struct {
	Calls  int   `json:"calls"`
	Ok     int   `json:"ok"`
	LastAt int64 `json:"lastAt"` // unix 秒
}

// File 状态文件形状（<DataRoot>/skill_stats.json）。
type File struct {
	Version int             `json:"version"` // 恒 1
	Skills  map[string]Stat `json:"skills"`
}

// StatView 绑定视图（camelCase 与状态文件同轴）。
type StatView struct {
	Name   string `json:"name"`
	Calls  int    `json:"calls"`
	Ok     int    `json:"ok"`
	LastAt int64  `json:"lastAt"`
}

// Record 纯函数累计：name trim 后空 → 原样返回；新名且已达 MaxSkills →
// 原样返回（宁少勿扰，不挤掉既有）；否则累计 calls、ok 命中加 Ok、LastAt=at。
// 传零值 File 时自动建表（调用方无需预判）。
func Record(f File, name string, ok bool, at int64) File {
	name = strings.TrimSpace(name)
	if name == "" {
		return f
	}
	if f.Skills == nil {
		f.Skills = make(map[string]Stat)
	}
	st, exists := f.Skills[name]
	if !exists && len(f.Skills) >= MaxSkills {
		return f
	}
	st.Calls++
	if ok {
		st.Ok++
	}
	st.LastAt = at
	f.Skills[name] = st
	f.Version = 1
	return f
}

// View 输出绑定视图：calls 降序 → name 升序稳定；恒非 nil。
func View(f File) []StatView {
	out := make([]StatView, 0, len(f.Skills))
	for name, st := range f.Skills {
		out = append(out, StatView{Name: name, Calls: st.Calls, Ok: st.Ok, LastAt: st.LastAt})
	}
	// 插入排序足够（条目数=技能数，个位到几十）；稳定：calls 相同按 name 升序。
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && statLess(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func statLess(a, b StatView) bool {
	if a.Calls != b.Calls {
		return a.Calls > b.Calls
	}
	return a.Name < b.Name
}
