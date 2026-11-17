package pantheon

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// 角色定义文件化（codex 蒸馏 T15，agent-roles 同构）：席位/角色从代码搬到
// 文件——用户在 `<pantheon 根>/agents/<role>.md` 放一个角色文件，众神殿即
// 多出一个模型席位；删文件即消失（席位表在装配期派生，不持久残留）。
//
// 文件形态（frontmatter + 正文即 persona）：
//
//	---
//	name: reviewer
//	description: 独立审稿人：对方案与文档做风险与漏洞审查，只读议事。
//	nicknames: 审稿人甲,审稿人乙
//	config_layer: reviewer.toml
//	---
//	（正文）你是众神殿的审稿人席位……
//
// 纪律（照抄上游）：description 是给模型看的 spawn 引导文案，必填非空——
// 空描述的文件拒绝加载并告警，不让一个坏文件炸全场；nicknames 可选（同一
// 角色多次入席时取不同昵称，防日志混淆）。

// RoleFile 是一个文件定义的席位角色。
type RoleFile struct {
	Name        string   // 席位 ID（frontmatter name 或文件名干；slug 形态）
	Description string   // spawn 引导文案（给模型看，必填）
	Persona     string   // 正文：该席位的系统人格（注入模型席位 system prompt）
	Nicknames   []string // 昵称候选（可选）
	ConfigLayer string   // 可选角色级配置层路径（相对角色文件目录）
	Source      string   // 来源文件路径（诊断/告警用）
}

// RoleOverrides 是角色级配置层的生效子集。指针区分「没有覆盖」与显式零值，
// 例如 max_steps=0 表示该席位不限步数，而不是回落到默认 12。
type RoleOverrides struct {
	MaxSteps      *int
	Temperature   *float64
	ContextWindow *int
}

type roleConfigLayer struct {
	Agent struct {
		MaxSteps      *int     `toml:"max_steps"`
		Temperature   *float64 `toml:"temperature"`
		ContextWindow *int     `toml:"context_window"`
	} `toml:"agent"`
}

// LoadRoleFiles 扫描 dir 下的 *.md 角色文件并解析。目录不存在返回空表
// （无角色文件是常态，不是错误）；单个文件不合法跳过并记入 warns（不炸全场）。
// 返回按 Name 排序的稳定序列。
func LoadRoleFiles(dir string) (roles []RoleFile, warns []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			warns = append(warns, fmt.Sprintf("read %s: %v", path, rerr))
			continue
		}
		role, perr := parseRoleFile(filepath.Base(e.Name()), string(data))
		if perr != nil {
			warns = append(warns, fmt.Sprintf("skip %s: %v", path, perr))
			continue
		}
		role.Source = path
		if _, perr := LoadRoleOverrides(role); perr != nil {
			warns = append(warns, fmt.Sprintf("skip %s: %v", path, perr))
			continue
		}
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Name < roles[j].Name })
	return roles, warns, nil
}

// parseRoleFile 解析单个角色文件：frontmatter（--- 围栏）取 name/description/
// nicknames，正文即 persona。description 必填非空；name 缺省用文件名干。
func parseRoleFile(filename, content string) (RoleFile, error) {
	role := RoleFile{}
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	body := content
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		lines := strings.Split(content, "\n")
		if idx := frontmatterEnd(lines); idx > 0 {
			fm, perr := parseRoleFrontmatter(lines[1:idx])
			if perr != nil {
				return role, perr
			}
			role = fm
			body = strings.Join(lines[idx+1:], "\n")
		}
	}
	if role.Name == "" {
		role.Name = stem
	}
	role.Name = strings.ToLower(strings.TrimSpace(role.Name))
	if role.Name == "" {
		return role, fmt.Errorf("席位名为空（frontmatter name 与文件名干皆空）")
	}
	if strings.ContainsAny(role.Name, " \t/\\") {
		return role, fmt.Errorf("席位名 %q 含非法字符（空格或路径分隔符）", role.Name)
	}
	role.Description = strings.TrimSpace(role.Description)
	if role.Description == "" {
		// 上游纪律：description 给模型当 spawn 引导，必填非空。
		return role, fmt.Errorf("description 必填（它是 spawn 引导文案，不是给人看的备注）")
	}
	role.Persona = strings.TrimSpace(body)
	return role, nil
}

// frontmatterEnd 返回闭合 --- 的行号（不含）；未闭合返回 -1（整文件当正文，
// 无 frontmatter 形态）。
func frontmatterEnd(lines []string) int {
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i
		}
	}
	return -1
}

// parseRoleFrontmatter 解析 frontmatter 的简单 KV（name/description/nicknames）。
func parseRoleFrontmatter(lines []string) (RoleFile, error) {
	role := RoleFile{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := cutKV(line)
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "name":
			role.Name = strings.TrimSpace(v)
		case "description":
			role.Description = strings.TrimSpace(v)
		case "nicknames":
			for _, n := range strings.Split(v, ",") {
				if n = strings.TrimSpace(n); n != "" {
					role.Nicknames = append(role.Nicknames, n)
				}
			}
		case "config_layer":
			role.ConfigLayer = strings.TrimSpace(v)
		}
	}
	return role, nil
}

// LoadRoleOverrides 读取角色级 TOML 配置层。只接受角色目录内的相对路径，
// 防止一条角色记录把任意本地配置拖进该席位。当前支持：
//
//	[agent]
//	max_steps = 6
//	temperature = 0.1
//	context_window = 64000
func LoadRoleOverrides(role RoleFile) (RoleOverrides, error) {
	var out RoleOverrides
	layer := strings.TrimSpace(role.ConfigLayer)
	if layer == "" {
		return out, nil
	}
	if filepath.IsAbs(layer) {
		return out, fmt.Errorf("config_layer 必须是相对角色文件目录的路径")
	}
	base := filepath.Dir(role.Source)
	path := filepath.Clean(filepath.Join(base, layer))
	baseReal, err := filepath.EvalSymlinks(base)
	if err != nil {
		return out, fmt.Errorf("解析角色目录: %w", err)
	}
	pathReal, err := filepath.EvalSymlinks(path)
	if err != nil {
		return out, fmt.Errorf("读取角色配置层 %s: %w", layer, err)
	}
	rel, err := filepath.Rel(baseReal, pathReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, `..`+string(filepath.Separator)) {
		return out, fmt.Errorf("config_layer 越出角色目录: %s", layer)
	}
	var raw roleConfigLayer
	if _, err := toml.DecodeFile(pathReal, &raw); err != nil {
		return out, fmt.Errorf("读取角色配置层 %s: %w", layer, err)
	}
	if raw.Agent.MaxSteps != nil && *raw.Agent.MaxSteps < 0 {
		return out, fmt.Errorf("config_layer %s: max_steps 不得为负", layer)
	}
	if raw.Agent.Temperature != nil && (*raw.Agent.Temperature < 0 || *raw.Agent.Temperature > 2) {
		return out, fmt.Errorf("config_layer %s: temperature 必须在 0..2", layer)
	}
	if raw.Agent.ContextWindow != nil && *raw.Agent.ContextWindow <= 0 {
		return out, fmt.Errorf("config_layer %s: context_window 必须大于 0", layer)
	}
	out.MaxSteps = raw.Agent.MaxSteps
	out.Temperature = raw.Agent.Temperature
	out.ContextWindow = raw.Agent.ContextWindow
	return out, nil
}

// cutKV 切 "key: value"（首个冒号）。
func cutKV(line string) (string, string, bool) {
	i := strings.IndexByte(line, ':')
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}
