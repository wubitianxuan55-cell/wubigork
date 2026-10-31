package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// buildInputManifest snapshots the start-time inputs of a council:
// brief evidence content and attachment files on disk. The stored brief is
// already persisted inside session.json, but attachment files live outside it
// and can be rewritten while the discussion is running — without the hash,
// next-day re-runs and path comparisons have no stable baseline
// (2026-10-31 裁决第三刀). Hash failures are recorded, never silent.
func buildInputManifest(req StartRequest) []InputDigest {
	digests := make([]InputDigest, 0, len(req.Evidence)+len(req.Attachments))
	for _, item := range req.Evidence {
		name := item.Name
		if name == "" {
			name = "未命名材料"
		}
		sum := sha256.Sum256([]byte(item.Content))
		digests = append(digests, InputDigest{
			Kind:   "evidence",
			Name:   name,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(item.Content)),
		})
	}
	for _, attachment := range req.Attachments {
		digest := InputDigest{
			Kind: "attachment",
			Name: attachment.Name,
			Path: attachment.Path,
		}
		data, err := os.ReadFile(attachment.Path)
		if err != nil {
			digest.Error = fmt.Sprintf("读取失败：%v", err)
			digests = append(digests, digest)
			continue
		}
		sum := sha256.Sum256(data)
		digest.SHA256 = hex.EncodeToString(sum[:])
		digest.Size = int64(len(data))
		digests = append(digests, digest)
	}
	if len(digests) == 0 {
		return nil
	}
	return digests
}

// verdictFingerprint binds a sign-off to the exact verdict version the user
// confirmed, so a forwarded deliverable can prove which edition it carries.
func verdictFingerprint(verdict *Verdict) string {
	if verdict == nil {
		return ""
	}
	data, err := json.Marshal(verdict)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// parentStanceHistoryLimit bounds how many previous-round statements a
// continuation carries into its brief (newest first), keeping the 24000
// prompt budget comfortable for the new brief itself.
const parentStanceHistoryLimit = 8

// parentStanceCardRunes is the mechanical excerpt length for one inherited
// statement — never a model re-summary, always annotated with original length
// and contribution ID.
const parentStanceCardRunes = 700

// parentStanceEvidence turns a parent session's discussion statements into
// stance-card evidence so a continuation can argue against historical points
// instead of only the six-item summary (2026-10-31 实弹反馈「历史论点继续
// 讨论」). Excerpts are mechanical and annotated; the brief never claims to
// inherit the full parent transcript.
func parentStanceEvidence(parent *Session) []EvidenceItem {
	if parent == nil {
		return nil
	}
	statements := make([]Contribution, 0, len(parent.Contributions))
	for _, contribution := range parent.Contributions {
		if contribution.Phase == PhaseSynthesis || strings.TrimSpace(contribution.Content) == "" {
			continue
		}
		statements = append(statements, contribution)
	}
	if len(statements) == 0 {
		return nil
	}
	// 最新发言优先：续议最需要承接的是收尾立场。
	if len(statements) > parentStanceHistoryLimit {
		statements = statements[len(statements)-parentStanceHistoryLimit:]
	}
	out := make([]EvidenceItem, 0, len(statements))
	for _, statement := range statements {
		runes := len([]rune(statement.Content))
		body := statement.Content
		if runes > parentStanceCardRunes {
			body = string([]rune(body)[:parentStanceCardRunes]) + fmt.Sprintf(
				"\n〔历史论点节选：原长 %d 字符，全文见父会话贡献 ID %s〕",
				runes, statement.ID,
			)
		}
		out = append(out, EvidenceItem{
			Name: fmt.Sprintf(
				"历史论点 · 上一场第 %d 轮 · %s（贡献 ID %s）",
				statement.Round, statement.AgentName, statement.ID,
			),
			Content: body,
		})
	}
	return out
}
