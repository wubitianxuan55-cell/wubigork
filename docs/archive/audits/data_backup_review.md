## 问题清单

| # | 严重度 | 文件:行 | 问题 | 建议 |
|---|--------|---------|------|------|
| 1 | 高 | internal/gaea/backup/backup.go:452-478 | **ApplyPending 部分失败后重试必失败且会破坏数据布局**。循环按条目逐个「先 rename(dst→before) 再 rename(src→dst)」，条目按字典序处理（Hephaestus.db→archive→config.toml→home-config→sessions→whisper_data）。若 whisper_data（最大、最易失败）的 rename 失败且 copyTree 中途失败（磁盘满/杀软锁文件），之前已消费的条目 src 已不在 staging。重试时这些条目 dst 存在→被移到新的 before 目录，src 不存在→rename/copyTree 均失败→永久卡死，且每次重试都把已应用数据再搬进新 before 目录。pending 永远清不掉。 | 主循环改为两阶段：第一阶段只把 dst 全部移到 before（无 src 依赖，可重入）；第二阶段对每个 src 存在才处理，src 缺失时视为「本条目已应用」跳过而非报错。或对每个条目记录已迁移状态，保证 apply 幂等可重入。补充「部分失败后重试成功」的测试。 |
| 2 | 高 | internal/gaea/backup/backup.go:480-495 | **home-config 永远不会恢复到 homeDir**。主循环（457-478 行）已把 stageDir/home-config 整体 rename 到 dataRoot/home-config，480 行才 os.Stat(stageDir/home-config)——必然不存在，整个 home 配置恢复块被跳过：~/.gaea_config.json 不恢复，反而在数据根留下垃圾 home-config/ 目录。 | 主循环里排除「home-config」（与 manifest.json 一样 continue），只让 480 行专门处理它（copy 到 homeDir + 备份旧配置）；或把 home 恢复挪到主循环之前。现有测试都没包含 home-config（backup_test.go:144-209 的 plan 无 HomeConfigRel 条目），补一条带 home-config 的用例。 |
| 3 | 高 | internal/gaea/backup/backup.go:246-257 与 144-152；internal/app/gaea_data_backup.go:25 | **快照连接无 busy_timeout + 静默回退复制缺 WAL，可产生缺失数据的「成功」备份**。snapshotSQLite 打开 file:...?mode=ro 未带 _busy_timeout（默认 0ms），而常驻连接（Hephaestus/hermes/chat/characterlib，均 busy_timeout=5000 的 WAL 模式，见 internal/gaea/db/database.go:45、internal/chat/db.go:35）在写入提交期间会让 VACUUM INTO 立刻返回 SQLITE_BUSY（仓库已有先例：herdsman_digitallife.go:102 对 ro 连接加了 _busy_timeout=5000）。失败后 Create 静默回退为原样复制 .db 文件，而 skip 规则排除了 -wal/-shm——WAL 未 checkpoint 的已提交数据必然丢失，且运行中复制还可能得到撕裂文件；manifest 无任何告警字段，用户拿到的是「成功」但缺数据的备份。 | DSN 加 &_busy_timeout=5000 并失败重试；回退路径改为 SQLite 在线备份 API（或先对源库做一次 wal_checkpoint(PASSIVE) 再复制 + 显式带上 -wal）；若必须回退，在 manifest 增加 warnings 字段并在前端/日志提示「快照失败，备份可能不完整」。 |
| 4 | 中 | internal/app/gaea_data_backup.go:204-215；internal/app/app.go:227-236；backup.go:424-505 | **恢复失败完全不可见**。.restore-result.json 只在成功路径写入（backup.go:502-503），失败路径虽设置 res.Error 但从不落盘；applyPendingRestore 又在打开 gaea.log 之前调用（app.go:228 vs 231），slog.Error 打到不可见 stderr。于是启动后前端只显示「有待应用的恢复」且永远失败（叠加 #1），用户无从得知失败原因；DataPanel 的失败告警分支（has_result && !applied）实际不可达。 | 失败路径也原子写 .restore-result.json（含 error）；或把 ApplyPending 挪到日志打开之后（数据库打开之前即可）；前端 pending 告警里附上上次失败原因。 |
| 5 | 中 | internal/app/gaea_data_backup.go:131-162 | **Restore 不检查已有 pending，可堆叠/覆盖**。已有 pending 时再次 GaeaDataBackupRestore 直接覆盖 pending 标记，旧 staging（可能 GB 级）成为孤儿目录永不清理（ClearPending 只删当前 staging）；同一秒内两次 Restore 会撞同名 .restore-stage-<ts> 目录，Extract 向已有目录合并解压，两版备份的独有文件混入同一 staging，应用出混合数据。 | Restore 前 ReadPending，存在时返回明确错误（「已有待应用恢复，请先取消」）；staging 名加随机后缀；Restore 成功后清理任何旧孤儿 .restore-stage-* 目录。 |
| 6 | 中 | internal/app/gaea_data_backup.go:72-104、218-232 | **GaeaDataBackupInfo 每次打开设置页全量递归统计 dirSize**，whisper_data 可达数百 MB~GB，同步阻塞 Wails 调用（每次进入「数据」页 + 备份/恢复后都触发），UI 卡顿。 | 改为异步（goroutine + 事件刷新）、按目录 mtime+大小缓存失效、或仅统计顶层条目并给 whisper_data 一个显示上限；Create 里复用的全量扫描可共享缓存。 |
| 7 | 中 | internal/gaea/backup/backup.go:8（注释）、DataPanel.tsx:195 | **宣称「失败可回滚」但无回滚代码**。失败/取消后 .restore-before-<ts> 目录保留旧数据，但 ApplyPending 与 GaeaDataBackupCancel 都没有从 before 恢复的路径，取消后数据根停留在部分替换状态，用户只能手动找回。 | 提供 GaeaDataBackupRollback（把 before 目录整体移回）或在 Cancel 时询问「同时回滚到恢复前数据？」；至少把文案改成「失败可手动找回（见 .restore-before-* 目录）」。 |
| 8 | 低 | internal/gaea/backup/backup.go:294-304 | safeZipRel 不拒绝盘符限定名。实测（Go filepath 验证）：../、..\、/abs、UNC（\\server\\share\\x）均被现有双重检查拦截，不会逃逸 destDir；但 C:/x、C:x 通过校验，会在 dest 内创建字面「C:」目录/文件（不逃逸，但属脏数据，且依赖 filepath.Join 当前行为）。 | 对 filepath.VolumeName(clean) != "" 显式拒绝；补 C:/x、反斜杠变体的 zip-slip 测试。 |
| 9 | 低 | frontend/src/components/settings/DataPanel.tsx:75-89 | 恢复按钮无二次确认（取消恢复反而有 Popconfirm），且 files.find(zip)||files[0] 在用户选到非 zip 时把任意文件传给后端，报错文案是「打开备份文件失败」而非提示选 zip。 | 恢复按钮加 Popconfirm（说明会替换当前数据、重启后生效）；选取后校验扩展名，非 zip 直接提示。 |
| 10 | 低 | internal/app/gaea_data_backup.go:114 | 备份文件名秒级时间戳：同一秒内两次备份互相覆盖（os.Create 截断），静默丢备份。 | 文件名加随机后缀或毫秒时间戳；dest 已存在同名 zip 时返回错误。 |
| 11 | 低 | internal/gaea/backup/backup.go:445-450 | .restore-before-* 每次成功恢复留下一整份旧数据（可能 GB 级）且无限累积、无清理策略。 | 保留最近 N 份（如 2），成功后清理更早的 before 目录；或在恢复结果提示里给出删除入口。 |
| 12 | 低 | internal/gaea/backup/backup.go:370-380 | WritePending 非原子（直接 WriteFile），Extract 完成与 pending 写入之间崩溃会留下孤儿 staging 目录。 | 临时文件+os.Rename 原子写；启动时清理无 pending 引用的 .restore-stage-* 孤儿目录。 |
| 13 | 低 | internal/gaea/backup/backup.go:317-351 | Extract 若先遇目录条目、后遇同名文件条目（或相反），OpenFile/读取会失败中止整个解压；zip 内无顺序保证。 | 解压前先收集全部条目：目录统一 MkdirAll，再按文件处理（目录条目直接跳过）；同名冲突取文件覆盖目录。 |
| 14 | 低 | internal/gaea/backup/backup.go:361-367、424-425 | PendingState.HomeDir 是死字段：ApplyPending 只用自己的 homeDir 参数，从不读 st.HomeDir；恢复前备份 home 配置也没有真正发生（#2 已覆盖主缺陷）。 | 删除该字段，或 ApplyPending 在 homeDir 为空时回退到 st.HomeDir。 |
| 15 | 低 | internal/gaea/backup/backup.go:58-70 | shouldSkip 的段前缀/后缀匹配过宽：任何文件名含 -wal/-shm/gaea.log 前缀或后缀（如 my-wal.md、gaea.log.backup）都会被误跳过。 | 跳过规则改为精确匹配 + 特定后缀（如 .db-wal$/.db-shm$），或至少只对 *.db-wal/*.db-shm 生效。 |
| 16 | 低 | internal/app/gaea_data_backup.go:91、166 | GaeaDataBackupInfo / GaeaDataBackupPending 吞掉 ReadPending 错误：pending 文件损坏时静默显示「无 pending」。 | 错误时返回 pending:false + pending_error 并前端提示。 |
| 17 | 低 | frontend/src/components/settings/DataPanel.tsx:49-50 | res as BackupInfo 只校验 data_root 是 string，entries 非数组（后端异常时）会在 info.entries.some 上 TypeError 崩溃。 | 增加 Array.isArray(entries) 校验或可选链兜底。 |

## 无问题项

- **A 一致性快照**：VACUUM INTO 在 WAL 未合并场景下数据完整性本身正确（测试验证 count=2），但 busy 失败与静默回退缺 WAL 是真实缺陷，见 #3。
- **B 恢复原子性**：staging 与数据根同卷、rename 为主、copyTree 为回退的思路可行；但重试幂等、home-config 覆盖、无回滚路径均有问题，见 #1/#2/#7。before 目录覆盖了全部被替换的顶层条目（同卷 rename），无遗漏。
- **C 路径安全**：实测确认 Windows 下 ../、..\、绝对路径、UNC 穿越均被 safeZipRel 拦截，目录/文件条目并存无顺序依赖（各自 MkdirAll），zip-slip 主体安全；盘符名与顺序冲突为低危，见 #8/#13。
- **D 性能**：dirSize 全量同步扫描为真实卡顿点，见 #6。
- **E 并发/幂等**：同秒 staging 撞名与已有 pending 时再次 Restore 的覆盖/孤儿泄漏真实存在，见 #5；重试幂等缺陷见 #1。
- **F 前端**：GaeaPickFiles 返回 FilePickResult{path,name,size} 与解构一致；zip 过滤与无确认问题见 #9；类型断言除 entries 非数组外安全，见 #17。
- **G 版本/契约**：AppVersion=2.20.0 与 wails.json productVersion、versioninfo.rc FileVersion/ProductVersion、CHANGELOG/releases/README 全部一致，无遗漏引用；bindings_office.go 六个 GaeaDataBackup* 门面签名与 gaea_data_backup.go 完全一致，生成的 OfficeB.d.ts 同步（GaeaPickFiles: Promise<Array<app.FilePickResult>>），无契约漂移。

（另：CharacterLibEditor 微信 beta 标注与 SettingsMobile「已冻结」标注为纯 UI 文案改动，未发现逻辑问题；SettingsMobile 的 QR 仍依赖 chart.googleapis.com 外部请求，移动端已冻结属低危遗留，未单列。）