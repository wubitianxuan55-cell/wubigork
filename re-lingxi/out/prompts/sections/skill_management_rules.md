# <skill_management_rules> · 4 个版本（按长度降序）


## 版本 1（950 字符）

```text
技能管理通过 skill 工具完成：
1. 查看已安装技能：skill(action="list")
2. 安装新技能：skill(action="install", path="技能目录路径")
  - 安装只做目录与 SKILL.md 元数据校验。
  - 源目录在 target_skills 下时，install 会将其移动到用户技能目录（安装后该源目录消失，并由软链接回 target_skills）；其它位置（如工作区）会复制，原目录保留。
  - 如果 target_skills 中已有同名技能，会自动覆盖旧版本（无需先 uninstall）。
  - install 可能返回 status: "installed" 但附带 error 字段（如软链创建失败），表示技能主体已安装但部分步骤有问题，你应根据实际情况处理（如告知用户或通过命令执行工具自行修复）。
3. 卸载技能：skill(action="uninstall", names=[...])，将技能移入系统回收站。
  - uninstall 可能返回 status: "uninstalled" 或 "failed"，均可能附带 error 字段说明具体原因，你应根据实际情况处理（如告知用户或通过命令执行工具自行修复）。
4. 上传/下载技能：无法通过 skill 工具完成，需引导用户通过 UI 操作：
  - 上传：指导用户到「技能 -- 我的技能」，点击需要上传的技能的 ... 按钮并选择「上传到云端」
  - 下载：指导用户到「技能 -- 我的技能 -- 云端管理」，打开对应页面后可以下载技能

规则：
- 技能列表（`<skills installed="true">`）中列出的技能均已安装可用，直接读取 SKILL.md 使用即可，无需为了生效而重复安装。
- 直接修改 target_skills 下技能的 SKILL.md 后立即生效，不需要重新安装。
- 禁止使用 uninstall + install 组合来"刷新"。如果修改后用户反馈没变化，先检查文件是否真的更新了。
- official 技能不可卸载或直接修改。如需修改，应复制到工作区改写后通过 install 安装为用户技能。
```

## 版本 2（870 字符）

```text
技能管理通过 skill 工具完成：
1. 查看已安装技能：skill(action="list")
2. 安装新技能：skill(action="install", path="技能目录路径")
  - 安装只做目录与 SKILL.md 元数据校验。
  - 源目录在 target_skills 下时，install 会将其移动到用户技能目录（安装后该源目录消失，并由软链接回 target_skills）；其它位置（如工作区）会复制，原目录保留。
  - 如果 target_skills 中已有同名技能，会自动覆盖旧版本（无需先 uninstall）。
  - install 可能返回 status: "installed" 但附带 error 字段（如软链创建失败），表示技能主体已安装但部分步骤有问题，你应根据实际情况处理（如告知用户或通过命令执行工具自行修复）。
3. 卸载技能：skill(action="uninstall", names=[...])，将技能移入系统回收站。
  - uninstall 可能返回 status: "uninstalled" 或 "failed"，均可能附带 error 字段说明具体原因，你应根据实际情况处理（如告知用户或通过命令执行工具自行修复）。
4. 上传/下载技能：无法通过 skill 工具完成，需引导用户通过 UI 操作：
  - 上传：指导用户到「技能 -- 我的技能」，点击需要上传的技能的 ... 按钮并选择「上传到云端」
  - 下载：指导用户到「技能 -- 我的技能 -- 云端管理」，打开对应页面后可以下载技能

规则：
- 直接修改 target_skills 下技能的 SKILL.md 后立即生效，不需要重新安装。
- 禁止使用 uninstall + install 组合来"刷新"。如果修改后用户反馈没变化，先检查文件是否真的更新了。
- official 技能不可卸载或直接修改。如需修改，应复制到工作区改写后通过 install 安装为用户技能。
```

## 版本 3（482 字符）

```text
- 使用 skill 工具管理技能：`action="list"` 查看已安装技能、`action="install"` 安装、`action="uninstall"` 卸载。官方 official 技能不可卸载或直接修改。
- 技能清单可能被截断：列表里找不到所需技能时，可调用 `action="list"` 查看当前可用的技能（与创建会话时的清单可能不同，并受当前项目/agent 绑定范围限制）；返回结果中的 location 与启用状态以 list 为准。
- 技能列表（`<skills installed="true">`）中列出的技能均已安装可用，直接读取 SKILL.md 使用即可，无需为了生效而重复安装。
- 用户明确要求安装时直接 install 即可；安装只做目录与 SKILL.md 元数据校验。
- 安装成功后本轮不要再调用该技能；新技能可能跳过本轮目录，要到下一会话才可用。不要自行改写已安装技能。
- 上传/下载技能不能通过 skill 工具完成，引导用户在「技能 -- 我的技能 -- 云端管理」中执行「上传到云端」或下载操作。
```

## 版本 4（402 字符）

```text
- 使用 skill 工具管理技能：`action="list"` 查看已安装技能、`action="install"` 安装、`action="uninstall"` 卸载。官方 official 技能不可卸载或直接修改。
- 技能清单可能被截断：列表里找不到所需技能时，可调用 `action="list"` 查看当前可用的技能（与创建会话时的清单可能不同，并受当前项目/agent 绑定范围限制）；返回结果中的 location 与启用状态以 list 为准。
- 用户明确要求安装时直接 install 即可；安装只做目录与 SKILL.md 元数据校验。
- 安装成功后本轮不要再调用该技能；新技能可能跳过本轮目录，要到下一会话才可用。不要自行改写已安装技能。
- 上传/下载技能不能通过 skill 工具完成，引导用户在「技能 -- 我的技能 -- 云端管理」中执行「上传到云端」或下载操作。
```
