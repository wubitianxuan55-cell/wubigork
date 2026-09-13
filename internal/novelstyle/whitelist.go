package novelstyle

// 书级白名单（oh-story T2 内核消费；gates.json 门禁 A 的 whitelist 口径）：
// <项目目录>/.deslop-whitelist，一行一个**显式授权片段**（# 开头为注释）。
// 命中白名单的 AI 味 issue 在打分与去味中豁免——作者明说过「这段就要这么写」。
// 无文件不建空表（gates.json 原文口径），文件读取尽力而为。

import (
	"os"
	"strings"
)

// LoadWhitelistFile 读书级白名单。文件不存在返回 (nil, nil)——调用方无需区分
// 「无文件」与「空表」，两者语义一致：无豁免。
func LoadWhitelistFile(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, line)
	}
	return entries, nil
}

// whitelisted 判断一个命中 span 的原文是否被授权：授权片段与命中文本互含
// （片段是 span 文本的子串，或 span 文本是片段的子串——作者常把整句抄进
// 白名单，而命中的可能是句中一个模式）。
func whitelisted(hit string, entries []string) bool {
	for _, e := range entries {
		if e == "" {
			continue
		}
		if strings.Contains(hit, e) || strings.Contains(e, hit) {
			return true
		}
	}
	return false
}

// ApplyWhitelist 打分豁免：摘除命中白名单的 issue，并按剩余 issue 重算分数
// （与 ScoreText 同一权重合成口径）。返回新分数；issues 原地过滤。
func ApplyWhitelist(score *TasteScore, text string, entries []string) int {
	if score == nil {
		return 0
	}
	if len(entries) == 0 {
		return score.Score
	}
	runes := []rune(text)
	kept := score.Issues[:0:0]
	for _, iss := range score.Issues {
		if iss.Start >= 0 && iss.End <= len(runes) && iss.Start < iss.End {
			if whitelisted(string(runes[iss.Start:iss.End]), entries) {
				continue
			}
		}
		kept = append(kept, iss)
	}
	score.Issues = kept
	n := 0
	for _, iss := range kept {
		n += severityToWeight(iss.Severity)
	}
	if n > 100 {
		n = 100
	}
	score.Score = n
	return n
}
