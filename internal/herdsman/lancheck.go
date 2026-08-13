// Package herdsman 提供对本地 Herdsman 桌面应用配置/状态的安全检测能力。
//
// 本文件实现 H0-4「LAN 暴露检测与告警」：逐行解析 herdsman config.yaml 的
// api 段（YAML 缩进风格，不使用 TOML 库），提取 lan_accessible 与 port，
// 暴露时给出结构化告警与中文处置指引。gaea 仅提示、绝不代改 herdsman 配置。
package herdsman

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// defaultPort 是 api.port 未配置时的缺省端口（本包共享常量，probe.go 亦引用）。
const defaultPort = 8080

// LanExposure 是 herdsman API 局域网暴露检测的结构化结果，直接供前端告警展示。
type LanExposure struct {
	// ConfigPath 被检查的配置文件路径。
	ConfigPath string `json:"config_path"`
	// Exposed 为 true 表示 api.lan_accessible=true，局域网暴露风险成立。
	Exposed bool `json:"exposed"`
	// Port 是 api.port 的值（未配置时缺省 8080）。
	Port int `json:"port"`
	// ConfigMissing 为 true 表示配置文件不存在（无法检测）。
	ConfigMissing bool `json:"config_missing"`
	// ParseError 非空表示配置存在但解析失败（非法布尔/端口等），或文件不可读。
	ParseError string `json:"parse_error,omitempty"`
	// Guidance 在 Exposed=true 时给出中文处置指引；其他情况为空。
	Guidance string `json:"guidance,omitempty"`
}

// CheckLanExposure 解析 configPath 指向的 herdsman config.yaml，返回 LAN 暴露检测结果。
//
// 解析规则（逐行、按 YAML 缩进风格）：
//   - 顶层键：行首无空白且非空行/注释行；
//   - api 段：以 "api:" 开头的顶层行开始，其后缩进更深（行首有空白）的行属于该段，
//     遇到下一个顶层键即结束；
//   - 段内提取 lan_accessible: true|false（缺省 false）与 port: <int>（缺省 8080）；
//   - lan_accessible 非布尔或 port 非整数视为解析失败（ParseError）。
func CheckLanExposure(configPath string) LanExposure {
	res := LanExposure{ConfigPath: configPath, Port: defaultPort}

	if configPath == "" {
		res.ConfigMissing = true
		res.ParseError = "未指定 herdsman 配置文件路径"
		return res
	}

	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			res.ConfigMissing = true
			res.ParseError = fmt.Sprintf("配置文件不存在：%s", configPath)
		} else {
			res.ParseError = fmt.Sprintf("读取配置文件失败：%v", err)
		}
		return res
	}
	defer f.Close()

	lanAccessible := false
	port := defaultPort

	scanner := bufio.NewScanner(f)
	inAPISection := false
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		// 空行与注释行不影响解析。
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 缩进判定：行首空白数。顶层键缩进为 0。
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		isTopLevel := indent == 0

		if isTopLevel {
			if strings.HasPrefix(trimmed, "api:") {
				// 进入（或重新进入）api 段。
				inAPISection = true
				continue
			}
			// 其他顶层键：若之前在 api 段，则段到此结束。
			inAPISection = false
			continue
		}

		// 缩进行：仅属于 api 段的行参与提取。
		if !inAPISection {
			continue
		}

		key, val, ok := lanSplitKeyValue(trimmed)
		if !ok {
			// 段内无法拆出 key: value 的行（如列表项）不影响检测。
			continue
		}
		switch key {
		case "lan_accessible":
			b, perr := lanParseBool(val)
			if perr != nil {
				res.ParseError = fmt.Sprintf("api.lan_accessible 值非法（%q）：%v", val, perr)
				return res
			}
			lanAccessible = b
		case "port":
			n, perr := strconv.Atoi(lanStripComment(val))
			if perr != nil {
				res.ParseError = fmt.Sprintf("api.port 值非法（%q）：%v", val, perr)
				return res
			}
			port = n
		}
	}
	if err := scanner.Err(); err != nil {
		res.ParseError = fmt.Sprintf("读取配置文件失败：%v", err)
		return res
	}

	res.Exposed = lanAccessible
	res.Port = port
	if lanAccessible {
		res.Guidance = lanBuildGuidance(configPath, port)
	}
	return res
}

// lanSplitKeyValue 把 "key: value" 拆成 key 与原始 value；key 只取段名本身。
// 无冒号或 key 为空视为不可解析（ok=false）。
func lanSplitKeyValue(line string) (key, val string, ok bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(line[idx+1:]), true
}

// lanParseBool 解析 YAML 布尔值：兼容 true/false 及 strconv.ParseBool 的
// 1/0/t/f 等写法，并剥离行内注释。
func lanParseBool(val string) (bool, error) {
	val = lanStripComment(val)
	switch strings.ToLower(val) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("无法解析为布尔值")
}

// lanStripComment 去掉标量值尾部的行内注释（" # 注释" 形式）。
func lanStripComment(val string) string {
	if idx := strings.Index(val, " #"); idx >= 0 {
		return strings.TrimSpace(val[:idx])
	}
	return strings.TrimSpace(val)
}

// lanBuildGuidance 生成中文处置指引：仅提示，不改 herdsman 配置。
func lanBuildGuidance(configPath string, port int) string {
	return fmt.Sprintf(
		"检测到 Herdsman API 已开启局域网访问（api.lan_accessible: true，端口 %d）且无鉴权："+
			"局域网内任意设备均可调用本机大模型，其 benchmark 等接口还能读取本机路径与运行端口，"+
			"存在数据泄露与算力滥用风险。gaea 无法代为修改 herdsman 配置，"+
			"请手动编辑 %s，将 api 段的 lan_accessible 改为 false 后重启 Herdsman 生效。",
		port, configPath)
}
