# <important> · 4 个版本（按长度降序）


## 版本 1（572 字符）

```text
- add 操作必须指定 mode（stdio/http）和 name
- stdio 模式必须指定 command（启动命令）
- http 模式必须指定 endpoint（服务 URL），可选 headers 用于鉴权。 
- remove/update/get 操作必须指定 id
- name 仅在 add 时必填
- 支持 list/get 查看所有类型的连接器，但 add/remove/update 仅支持编辑 custom 类型的连接器
- 连接器必须有完善的description 才能了解何时使用这个连接器。 因此， 当用户仅仅给出一个endpoint/ 绝对路径，希望添加连接器时， 你应当按照如下步骤进行：
	1. 通过add 简略的添加， 这一步重心在正确填写endpoint、headers、command等配置， 保证连接器能正常连通； 对于command， 可以先使用当前可用的命令执行工具来探测其正确性。
	2. 调用 connector_exec 工具， 通过 list action 来探查连接器提供的能力
	3. 根据上一步探查出的能力， 更新连接器的name 和 description.
- 当用户给出了完整的description时, 应当不做任何修改的、原样使用用户提供的description。
```

## 版本 2（206 字符）

```text
- 必须先通过 connector_manage 工具确认连接器已注册并获取其 ID
- call 操作时 tool 参数必填，arguments 参数根据工具定义可选或必填
- 返回结果由 MCP 服务决定，可能为 JSON 或纯文本
- 部分连接器可能会出现session过期等问题。 出现时， 进行一次尝试即可。 如果重试无法解决问题， 则不应当继续盲目重试， 而应当整合已知信息， 将问题分析汇报给用户。
```

## 版本 3（137 字符）

```text
- install 时 path 参数传入技能目录的完整路径。源目录在 target_skills 下时会移动；其它位置会复制并保留原目录。
- uninstall 时 names 参数传入技能目录名（非完整路径）。
- source 为 official 的技能不可卸载。
```

## 版本 4（123 字符）

```text
⚠️ **调用此工具后必须完成交付**：
- creative 的 tool result 不会展示给用户，用户看不到生成的内容
- 调用后你必须将创作内容实际交付给用户（直接输出或写入文件）
- 禁止只回复"已为你生成了..."而不展示实际内容
```
