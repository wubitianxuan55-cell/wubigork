# gaea Day Log · 2026-11-19 · 办公板块工具 R2 分档诚实恢复契约（众神殿 pan_1795058214142 裁决施工）

> 众神殿 pan_1795058214142（「优化完善办公板块的工具，特别是本地工具」，4 席 3 轮，
> 主席 Codex，用户签注 accepted）裁决定案：办公本地工具的主要矛盾已从数量不足转为
> **COM 直接写入后的恢复边界与可验证性**；唯一 P1=R2 分档诚实恢复契约（影响高、
> 成本低）；R1 持久快照另立需求（须先定义未保存修改语义）；打开中 xlsx 的占用 A/B
> 后置观察。本批施工 action-1（R2 契约）+ action-4（文档回写）。

## 1. 裁决事实基础（各席工具核验）

- 现行 office_edit 描述只承诺 Word 一档「一次 Ctrl+Z 可整体撤销」（Excel/PPT 无说明）；
  office_save 描述只说证据卡「可复核」——**是边界遗漏，不是虚假承诺，R2 是补齐不是删除**。
- `OfficeHelper.cs` 的 undo 命令（:925 非 word 直接 BAD_ARGS）与 UndoRecord
  （:9741 StartCustomRecord/:9801 EndCustomRecord 仅 word 分支）——Excel/PPT 无整批撤销。
- office_save Journal 卡 BaselinePath 恒空+摘要自陈「无基线快照」（诚实缺省已实现）。
- Excel/PPT 应用自身撤销粒度未核验——只能声明助手不提供整批撤销，**不得写成逐操作可撤**
  （grok-r3 推翻 dsh-r3「逐操作」措辞：用新的过度声明替换旧的省略）。

## 2. R2 契约落地（action-1）

- **office_edit 描述分档**（`internal/app/office_com_tools.go`）：「Word 的一批修改…」
  单句扩为分档句——Word=一次 Ctrl+Z 整批+仅限当前文档与当前会话+不能恢复编辑前版本；
  Excel/PPT=助手不提供整批撤销+应用自身撤销能力未核验不要向用户保证可撤销。
- **office_save 描述**：末句「落盘后自动登记证据卡（可复核）。」→「（仅可复核，无基线
  快照——保存后没有跨会话回滚）。」
- **回执侧单源边界行**（`internal/office/comhelper/format.go`）：新导出
  `UndoBoundary(kind AppKind) string`（Word/Excel·PPT 两档文案）；`FormatEdit`
  签名加 kind 参数，成功与失败两条支路末尾都追加边界行（grok-r3 定稿：成功/失败回执
  都必须带分档失效边界）；唯一调用方 office_com_tools.go:842 同步。
- **save 回执恒带恢复边界**（office_com_tools.go text 组装处）：「恢复边界：证据卡仅可
  复核、无基线快照，保存后没有跨会话回滚——改错只能靠应用内撤销（仅 Word 一次 Ctrl+Z
  整批、限当前会话）或重做。」带不带 Final check 段的保存回执都携带。
- **静态契约测试**（裁决验收=成功/失败/保存三类测试均证明分档边界）：
  - `internal/office/comhelper/format_edit_boundary_test.go`：FormatEdit 成功/失败×
    word/excel/ppt 象限（含 prechecked 码 BAD_ARGS/NOT_FOUND 失败腿）边界断言+禁词
    「可回滚/可恢复」+Excel·PPT 不得出现「逐操作撤销」。
  - `internal/app/office_recovery_contract_test.go`：①描述钉住（两描述含分档钉句、
    禁词「可回滚/可恢复」）；②save 回执恢复边界四钉句+Journal 诚实缺省上锁
    （BaselinePath==""+摘要含「无基线快照」）；③源级守卫 TestOfficeHelperUndoGuardPinned
    （repoRoot 先例=wire_shape_guard_test.go）：读 OfficeHelper.cs 钉 undo 守卫+
    StartCustomRecord/EndCustomRecord 三行原文——未来给 Excel/PPT 加整批撤销时红=
    提醒同批更新 R2 文案（dsh-r3 断言④）。
- **明确不动**：office_live_prompt.go（A/B 出现可复现失败才改提示词=裁决
  disagreements #2）；OfficeHelper.cs 行为（R2 纯文案+测试面）。

## 3. registrymap 谓词+灵犀文档回写（action-4）

- `internal/gaea/tool/registrymap.go` xlsx_apply 谓词补「落盘前整文件基线快照+证据卡」
  （对齐 docx_apply/docx_accept/pptx_apply 同族句式，防基线实现被误判缺失）；
  镜像文档 `docs/gaea-tool-registry-map-2026-11.md` 同步。
- `docs/gaea-lingxi-optimize-plan-2026-11.md` 文首回写（grok-r2 裁定）：P1 四项
  2026-11-13 收官（复用系统 Python）+P1-4 dock 同日晚撤项（ec3f4f48）+batch2 五项
  同日收官+权威=todos/batch2、勿重开；L163 阻塞点行标注已解除；L130-133 P1-4 残留
  步骤标「仅存档勿执行」。
- 顺手修 HEAD 既有假红：9e366084（子代理线）给 registrymap 加了 send_message 行但
  镜像文档漏导出——TestRegistryMapDocCoversMap 红，补文档行即绿。

## 4. 验证与装机

- go test：comhelper 9.4s 绿+tool 1.9s 绿+tool/builtin 19.6s 绿+app 全包 119.9s 绿；
  gofmt（本批文件）清；CGO_ENABLED=0 全树编译过；golangci 本批文件 0 issues
  （capability.go 等 9 处 gofmt+gaea_verify.go rollbackRecord unused=各线既有非本批）。
- 构建：frontend npm build 29s→wails build 59.9s（ucrt64/CGO_ENABLED=1）→
  smoke.ps1 /api/health 200。
- 装机：gaea_install.ps1 INSTALL OK（exe sha256 9B3761998EBB…），旧壳已杀净
  （CIM probe：no gaea.exe running）。

## 5. 余项（裁决后置，勿擅自提前）

- **action-2 打开中 xlsx 观察 A/B**：R2 落地后真机跑（Excel 与 WPS 各 ≥3 次关/开态
  对照，记录原始回执+模型两步内动作；反例金标=静默半写立即升级）；zcode-r3 七步设计
  在最终文档。**先 A/B 后改提示词**，不得倒置。
- **action-3 R1 保存前快照**：另立需求，用户先定义「未保存修改」语义（有改动/无改动/
  大文件三例）再评估实现。
- Excel/PPT 应用自身撤销粒度真机演练（临时文件单批修改+逐次 Ctrl+Z 比对，约 1 小时/应用）。
