package ai

import "sort"

// ComfyUITxt2imgModels 返回已登记的文生图模型族（稳定字典序）。
// 绘梦导演（11.6）的模型候选集从这里取——txt2imgWorkflows 的单一事实源
// 导出口，app 层不抄表（新增模型登记后导演候选集自动跟上）。
func ComfyUITxt2imgModels() []string {
	out := make([]string, 0, len(txt2imgWorkflows))
	for m := range txt2imgWorkflows {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
