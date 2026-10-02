import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\round-1-fixes.md'
s = io.open(p, encoding='utf-8', newline='').read()

add = '''5. **集成门实测（收尾）**：本地快闸 `scripts/ci.ps1` 全程通过——version-drift / clean-tmp / `go build` / `go vet` / golangci（钉版 v2.14.0）/ `go test ./... -count=1` / 前端 eslint + build + 全量 vitest（**414 文件 / 3596 例**）/ E 系守卫 / check-docs，脚本自身以 `CI OK` 收尾。外层管道返回的 exit 1 来自 node 的 stderr 警告被 PowerShell 记为 `NativeCommandError`（本仓在册的 stderr 陷阱，见 `ci.ps1` 头部注释），故主代理**直接重跑**了两项关键闸复核：`go test ./... -count=1` → exit 0、0 条 FAIL；`golangci-lint run ./...` → **0 issues**。
6. **门禁抓到的问题（教训：子代理自测绿 ≠ 接门禁绿）**：首轮快闸在 golangci 挂掉——线1 新文件 `render_preserve.go` 留了 2 个**未被使用**的辅助函数（`blockStringValue`、`keyLine.strValue`）与 1 个只写不读的结构字段 `val`；主代理清理后重跑才全绿。**作业书应加一条「无用代码自检」**（新建文件里不得留只被自己引用的辅助函数/只写字段），否则每轮都会有这样的收尾返工。
7. **本轮不做绑定面/生成物/版本**：未改 `bindingNames.ts`、`bridge/**`、`wailsjs/**`、`frontend/**`（线5 实测前端无需改动）、未抬版本、未动 CHANGELOG 与 AGENTS 速览——发版仪式归用户；`scripts/check-primitives.mjs` 仍是 warn 档，接 CI 前需拍板验收线。
'''

anchor = '4. 并行线的编译瞬时态会互相影响'
if anchor in s:
    # 在最后一条后面追加
    idx = s.find(anchor)
    end = s.find('\n', idx)
    s = s[:end + 1] + add + s[end + 1:]
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
    print('appended')
else:
    print('MISS anchor')
