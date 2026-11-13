# <examples> · 1 个版本（按长度降序）


## 版本 1（659 字符）

```text
创建周期性任务：
- 每天早上 9 点: action="add", schedule="0 0 9 * * *"
- 工作日早上 9 点: action="add", schedule="0 0 9 * * 1-5"
- 每 2 小时: action="add", schedule="0 0 */2 * * *"
- 周一三五下午 2 点: action="add", schedule="0 0 14 * * 1,3,5"

创建一次性任务：
- 指定时间执行一次: action="add", schedule="2026/3/31 18:26:03"

查看任务列表：
- action="list"

修改任务：
- 修改执行时间: action="update", task_id="xxx", schedule="0 0 10 * * *"
- 修改任务内容: action="update", task_id="xxx", prompt="新的任务描述"
- 启用任务: action="update", task_id="xxx", enabled=true
- 禁用任务: action="update", task_id="xxx", enabled=false
- 同时修改内容并禁用: action="update", task_id="xxx", prompt="新的任务描述", enabled=false

删除任务：
- action="delete", task_id="xxx"
```
