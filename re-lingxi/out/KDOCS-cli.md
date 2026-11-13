# kdocs-cli.exe（静态盘点）

- 二进制: `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe`（7.3 MB，**Go 1.23.3**，module `kdocs-cli` `(devel)`，仅 3 个依赖 → 纯 std 客户端）
- sha256 `24b3e7865d4b1eccbb893868e7ee7384e5ae94a08c4d3c234bc6ab41a74fae28`

## CLI 形状（Usage 原文）

- `Usage: %s auth set-token <token> token persist env fallback failed %s %s %s <操作类型> --help 修复完成，请重试命令。Checksum verification failed: %v bytes.Buffe`
- `Usage: %s %s <action> [json_params] Error: unknown action %q in service %q http: putIdleConn: keep alives disabledusername/password authenticat`
- `Usage: %s call <tool_name> [json_params] /office/v5/ai/skill_hub/wps_auth/exchangehttp2: no cached connection was availablehttp2: Transport health`
- `Usage: %s feedback [--reason <trigger_reason>] [--skill-name <name>] ⚠ %s v%s available (current: v%s). Run '%s upgrade -y' to update. http2: Tra`
- `Usage: %s <service> <action> [json_params] %s auth <subcommand> { "file_id": "f_abc123", "page_size": 20, "page_token": "NEXT_PAGE_TOKEN`
- `Usage: %s auth <subcommand> Subcommands: set-token <token> 将 Token 保存到系统密钥链（推荐，用户已有 Token 时使用） { "file_id": "string", "worksheet_id": 1,`
- `Usage: %s jsapi wiki --file <params.json|-> 参数 JSON: {"component":"et|wps|wpp|pdf","queries":[{"title":"...","keywords":["..."]}]} Wiki 数据目录`
- `Usage: %s upgrade [flags] Self-upgrade %s to a newer version. Flags: --check Check for updates without installing --version <ver>`

## 端点（路径原文）

- `/api.wps.cnppt/references/...pdf/references/...`
- `/api/v1/tool`
- `/kdocs-auth/auth-guide`
- `/kdocs-cli/auth.go`
- `/kdocs-cli/auth_refresh.go`
- `/kdocs-cli/client.go`
- `/kdocs-cli/client_id.go`
- `/kdocs-cli/doctor.go`
- `/kdocs-cli/env_windows.go`
- `/kdocs-cli/feedback.go`
- `/kdocs-cli/help.go`
- `/kdocs-cli/hooks.go`
- `/kdocs-cli/input.go`
- `/kdocs-cli/jsapi.go`
- `/kdocs-cli/jsapi_wiki.go`
- `/kdocs-cli/keyring.go`
- `/kdocs-cli/keyring_mid_windows.go`
- `/kdocs-cli/main.go`
- `/kdocs-cli/meta_generated.go`
- `/kdocs-cli/request_source_enc.go`
- `/kdocs-cli/sid_fingerprint.go`
- `/kdocs-cli/trace.go`
- `/kdocs-cli/update_notice.go`
- `/kdocs-cli/upgrade.go`
- `/kdocs-cli/urls.go`
- `/kdocs-cli/variant.go`
- `/kdocs/mobile/touch/apple-152.png`
- `/office/v5/ai/skill_hub/`
- `/office/v5/ai/skill_hub/wps_auth/exchangehttp2`
- `/v2.runtime`

## 动作/参数词表

- snake_case 候选 **1293** 个（动作名与入参名混在同一张字符串表，静态无法完全区分）
- 明确是动作的样本：`create_doc`、`block_insert`、`block_delete`、`block_query`、`block_update`、`add_fields`、`add_calculated_field`、`add_chart`、`ap_file`、`create_records`

## 与 gaea 的关系

灵犀把云文档能力做成 **独立 CLI（`kdocs-cli <service> <action> [json_params]`）**，
由 `KDOCS_CLI_PATH` + `WPS_SID` 驱动，技能脚本只做参数拼装。gaea 若不接金山云，
这一层整体不移植；但「云能力=独立 CLI、技能只拼参数」这个**分层**值得照搬（比把云 SDK 编进主进程轻）。