package contextview

import (
	"encoding/json"
	"strconv"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// DnaAt 在一次日志遍历中折叠多个历史 request_header 时点的趋势 DNA 带。
// 输入顺序原样返回；非 request_header/未知 seq 返回 Ok=false，不用 live
// surface 顶替。节点/归档窗口与 SurfaceAt 共用 FoldTimeline 上限。
func DnaAt(entries []session.LogEntry, seqs []int64, window int64) []DnaSample {
	targets := make(map[int64]struct{}, len(seqs))
	for _, seq := range seqs {
		if seq > 0 {
			targets[seq] = struct{}{}
		}
	}
	found := make(map[int64]DnaSample, len(targets))
	if len(targets) > 0 {
		f := &folding{window: window}
		for _, entry := range entries {
			f.apply(entry)
			if _, wanted := targets[entry.Seq]; !wanted || entry.Kind != session.KindRequestHeader {
				continue
			}
			f.trimBrowserWindow()
			var header requestHeaderPayload
			_ = json.Unmarshal(entry.Payload, &header)
			toolBands := make([]DnaBand, 0, len(header.Tools))
			toolNames := map[string]int{}
			for _, tool := range header.Tools {
				if tool.Name == "" {
					continue
				}
				tokens := estimateTokens(tool.Schema)
				if tokens <= 0 {
					continue
				}
				toolNames[tool.Name]++
				key := "tool:" + tool.Name
				if toolNames[tool.Name] > 1 {
					key += "#" + strconv.Itoa(toolNames[tool.Name])
				}
				toolBands = append(toolBands, DnaBand{
					Key: key, Cat: catTools, Tokens: tokens,
					NodeSeq: entry.Seq, Label: tool.Name,
				})
			}
			found[entry.Seq] = DnaSample{
				Ok:    true,
				Seq:   entry.Seq,
				Bands: dnaBandsOf(f.nodes, toolBands),
			}
		}
	}
	out := make([]DnaSample, 0, len(seqs))
	for _, seq := range seqs {
		sample, ok := found[seq]
		if !ok {
			sample = DnaSample{Seq: seq, Bands: []DnaBand{}}
		}
		if sample.Bands == nil {
			sample.Bands = []DnaBand{}
		}
		out = append(out, sample)
	}
	return out
}

func dnaBandsOf(nodes []SurfaceNode, toolBands []DnaBand) []DnaBand {
	lastSystem := -1
	lastTools := -1
	for i, node := range nodes {
		switch node.Cat {
		case catSystem:
			lastSystem = i
		case catTools:
			lastTools = i
		}
	}
	bands := make([]DnaBand, 0, len(nodes))
	if lastSystem >= 0 && nodes[lastSystem].Tokens > 0 {
		node := nodes[lastSystem]
		bands = append(bands, DnaBand{
			Key: "sys", Cat: catSystem, Tokens: node.Tokens, NodeSeq: node.Seq,
		})
	}
	if len(toolBands) > 0 {
		bands = append(bands, toolBands...)
	} else if lastTools >= 0 && nodes[lastTools].Tokens > 0 {
		node := nodes[lastTools]
		bands = append(bands, DnaBand{
			Key: "tools", Cat: catTools, Tokens: node.Tokens, NodeSeq: node.Seq,
		})
	}
	for _, node := range nodes {
		if node.Cat == catSystem || node.Cat == catTools || node.Tokens <= 0 {
			continue
		}
		label := node.Tool
		if label == "" {
			label = node.Name
		}
		bands = append(bands, DnaBand{
			Key:     "n" + strconv.FormatInt(node.Seq, 10),
			Cat:     node.Cat,
			Tokens:  node.Tokens,
			NodeSeq: node.Seq,
			Label:   label,
		})
	}
	return bands
}
