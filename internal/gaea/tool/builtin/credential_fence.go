package builtin

// 凭据围栏（B1，dsh-ops credential-policy 蒸馏，2026-11-10）：结构化读面
// （read_file/grep）对凭据形态路径 deny。bash 不围（归审批/PublishGuard 域，
// 如实记档）；ls 不围（文件名非内容）。规则=目录段+名字双轨，大小写不敏感；
// 刻意保守（office 文档永不命中）。误拦代价=模型收到结构化指引转用户操作；
// 漏拦代价=秘密进上下文——按 fail-closed 取向。

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// credentialDirSegments 目录段围栏（路径任一段命中即拦）。
var credentialDirSegments = map[string]bool{
	".ssh": true, ".aws": true, ".kube": true, ".gnupg": true,
	".docker":        true, // config.json 含 registry 凭据
	".config/gcloud": true, ".azure": true,
}

// credentialNameRules 名字围栏（base 名小写精确或后缀）。
var credentialNameExact = map[string]bool{
	".env": true, ".netrc": true, ".git-credentials": true,
	"id_rsa": true, "id_ed25519": true, "id_ecdsa": true, "id_dsa": true,
}

var credentialNameSuffixes = []string{
	".pem", ".key", ".pfx", ".p12", ".ppk", ".keystore",
}

// credentialFenceRule 报告路径命中的围栏规则；未命中返回 ("", false)。
func credentialFenceRule(p string) (string, bool) {
	p = strings.TrimPrefix(filepath.ToSlash(p), "/")
	if p == "" {
		return "", false
	}
	lower := strings.ToLower(p)
	segs := strings.Split(lower, "/")
	for _, seg := range segs {
		if credentialDirSegments[seg] {
			return "目录段 ." + strings.TrimPrefix(seg, ".") + "/", true
		}
		// .env 家族（.env / .env.local / .env.production）
		if seg == ".env" || strings.HasPrefix(seg, ".env.") {
			return "名字 " + seg, true
		}
	}
	base := segs[len(segs)-1]
	if credentialNameExact[base] {
		return "名字 " + base, true
	}
	for _, suf := range credentialNameSuffixes {
		if strings.HasSuffix(base, suf) {
			return "后缀 " + suf, true
		}
	}
	return "", false
}

// credentialDenied 结构化拒绝（错误即值：模型收到可自愈指引）。
// B2：升 tool.Failure{denied}——围栏先于读取生效，effect none；原样重试
// 恒被拒故不可重试。文案前缀保留为兼容层。
func credentialDenied(toolName, p, rule string) error {
	return &tool.Failure{
		Code: tool.CodeDenied,
		Err: fmt.Errorf("CREDENTIAL_DENIED: 凭据围栏拒绝 %s 读取 %s（命中规则 %s）。"+
			"该文件疑似凭据/密钥，读取会把它带进对话上下文；如确需访问请用户自行操作或明确授权后调整围栏。", toolName, p, rule),
		EffectState: tool.EffectNone,
		NextAction:  "凭据文件不进上下文：确需访问请用户操作或明确授权",
	}
}
