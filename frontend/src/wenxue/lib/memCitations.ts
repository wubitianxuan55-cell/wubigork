// memCitations — 记忆引用键识别（对应 Go 侧 internal/gaea/memory/citations.go
// 的同一正则语义）：模型在回答中标注 [MEM:<name>]（记忆注入块教过的格式），
// AST 层把它转换成 mem: 链接节点，由 Markdown 的 a 渲染器渲染成可点击溯源
// 徽标（点击弹层展示记忆详情与沉淀来源——「你引用的资料是不是真的」闭环）。

export interface MemCitation {
  start: number;
  end: number;
  name: string;
}

const MEM_CITATION_RE = /\[mem:([a-z0-9][a-z0-9-]*)\]/gi;

/** 在一段文本里找出全部引用键（含起止偏移，name 小写归一）。 */
export function findMemCitations(value: string): MemCitation[] {
  const out: MemCitation[] = [];
  MEM_CITATION_RE.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = MEM_CITATION_RE.exec(value)) !== null) {
    out.push({ start: m.index, end: m.index + m[0].length, name: m[1].toLowerCase() });
  }
  return out;
}
