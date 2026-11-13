# comhelper 上游 provenance

- 来源：https://github.com/copylee711/dsh-office v1.2.0（commit `2069af2d6b2b2a9819321a73ea3427716b686608`，2026-10-23）
- 本目录 `OfficeHelper.cs` = 上游 `helper/OfficeHelper.cs` **逐字节拷贝**（`cmp` 验证一致）。
  升级上游时：重新拷贝 → 跑本包协议测试 → 真机冒烟；不要在本文件上做本地修改
  （改动会破坏与上游的 diff 能力，偏离一律写在 Go 侧或本文档）。
- `UPSTREAM-LICENSE` = 上游 MIT 许可证全文（OfficeHelper.cs 的授权依据）。
- 本地快照：`.tmp/dsh-office-upstream-src`（调研批克隆，见 docs/gaea-dsh-office-distill-research-2026-11.md）。
- 已知偏离见 doc.go（/codepage:65001、波1 无 math 腿、体验开关硬编码）。
