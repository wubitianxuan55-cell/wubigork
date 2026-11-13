# 灵犀主提示词骨架（按二进制顺序）

- 源: `C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe`，区间 `40380000`–`41070000`
- 已知段 131 个；该区间内段出现 141 次（压缩连续重复后 130 个节点）
- 读法：这是**装配顺序**（模板 include 展开后的样子），段正文见 `prompts/sections/<tag>.md`

## 装配顺序

| # | offset | 段 |
|---|---|---|
| 1 | `40405360` | `<group_chat>` |
| 2 | `40405719` | `<skills>` |
| 3 | `40406507` | `<runtime>` |
| 4 | `40407026` | `<language>` |
| 5 | `40407199` | `<format>` |
| 6 | `40407612` | `<im_scene>` |
| 7 | `40407724` | `<agent_loop>` |
| 8 | `40407737` | `<tool_call_style>` |
| 9 | `40408209` | `<agent_loop>` |
| 10 | `40408341` | `<memory_system>` |
| 11 | `40408362` | `<memory_index>` |
| 12 | `40408381` | `<memory_recall_rules>` |
| 13 | `40408927` | `<tool_use>` |
| 14 | `40409081` | `<system_reminder>` |
| 15 | `40409130` | `<tool_call_style>` |
| 16 | `40409540` | `<assistant_behavior>` |
| 17 | `40410428` | `<error_handling>` |
| 18 | `40410984` | `<search_strategy>` |
| 19 | `40411332` | `<assistant_reminders>` |
| 20 | `40411414` | `<system_reminder>` |
| 21 | `40411545` | `<timer_task_restriction>` |
| 22 | `40411684` | `<cloud_environment_restriction>` |
| 23 | `40411862` | `<safety>` |
| 24 | `40412161` | `<delete_file_rules>` |
| 25 | `40412939` | `<ask_user_question_tool>` |
| 26 | `40413409` | `<render_widget_tool>` |
| 27 | `40413524` | `<file_handling_rules>` |
| 28 | `40414245` | `<skill_management_rules>` |
| 29 | `40415336` | `<sharing_files>` |
| 30 | `40416409` | `<response_ending>` |
| 31 | `40416887` | `<disclosure_prohibition>` |
| 32 | `40417549` | `<user_profile>` |
| 33 | `40418492` | `<runtime>` |
| 34 | `40419006` | `<language>` |
| 35 | `40419115` | `<format>` |
| 36 | `40419425` | `<im_scene>` |
| 37 | `40419535` | `<agent_loop>` |
| 38 | `40419548` | `<tool_call_style>` |
| 39 | `40420432` | `<agent_loop>` |
| 40 | `40420512` | `<memory_system>` |
| 41 | `40420533` | `<memory_index>` |
| 42 | `40420552` | `<memory_recall_rules>` |
| 43 | `40420990` | `<tool_use>` |
| 44 | `40421266` | `<system_reminder>` |
| 45 | `40421315` | `<tool_call_style>` |
| 46 | `40421711` | `<lingxi_behavior>` |
| 47 | `40422589` | `<lingxi_reminders>` |
| 48 | `40422672` | `<system_reminder>` |
| 49 | `40422748` | `<error_handling>` |
| 50 | `40423219` | `<safety>` |
| 51 | `40423518` | `<delete_file_rules>` |
| 52 | `40424009` | `<ask_user_question_tool>` |
| 53 | `40424500` | `<render_widget_tool>` |
| 54 | `40424605` | `<search_strategy>` |
| 55 | `40424982` | `<timer_task_restriction>` |
| 56 | `40425120` | `<cloud_environment_restriction>` |
| 57 | `40425329` | `<file_handling_rules>` |
| 58 | `40426197` | `<skill_management_rules>` |
| 59 | `40427331` | `<sharing_files>` |
| 60 | `40429987` | `<response_ending>` |
| 61 | `40430464` | `<disclosure_prohibition>` |
| 62 | `40430947` | `<user_profile>` |
| 63 | `40431428` | `<group_chat>` |
| 64 | `40431787` | `<skills>` |
| 65 | `40432575` | `<runtime>` |
| 66 | `40433094` | `<language>` |
| 67 | `40433267` | `<format>` |
| 68 | `40433680` | `<im_scene>` |
| 69 | `40433792` | `<agent_loop>` |
| 70 | `40433805` | `<tool_call_style>` |
| 71 | `40434277` | `<agent_loop>` |
| 72 | `40434409` | `<memory_system>` |
| 73 | `40434430` | `<memory_index>` |
| 74 | `40434449` | `<memory_recall_rules>` |
| 75 | `40434995` | `<tool_use>` |
| 76 | `40435271` | `<system_reminder>` |
| 77 | `40435320` | `<tool_call_style>` |
| 78 | `40435730` | `<assistant_behavior>` |
| 79 | `40436618` | `<error_handling>` |
| 80 | `40437174` | `<search_strategy>` |
| 81 | `40437522` | `<assistant_reminders>` |
| 82 | `40437604` | `<system_reminder>` |
| 83 | `40437735` | `<timer_task_restriction>` |
| 84 | `40437874` | `<cloud_environment_restriction>` |
| 85 | `40438052` | `<safety>` |
| 86 | `40438380` | `<web_artifact>` |
| 87 | `40439307` | `<delete_file_rules>` |
| 88 | `40440085` | `<ask_user_question_tool>` |
| 89 | `40440555` | `<render_widget_tool>` |
| 90 | `40440670` | `<file_handling_rules>` |
| 91 | `40441775` | `<skill_management_rules>` |
| 92 | `40442946` | `<sharing_files>` |
| 93 | `40444046` | `<response_ending>` |
| 94 | `40444524` | `<disclosure_prohibition>` |
| 95 | `40445186` | `<user_profile>` |
| 96 | `40446026` | `<noop>` |
| 97 | `40450084` | `<rollout_summary>` |
| 98 | `40450170` | `<noop>` |
| 99 | `40458308` | `<noop>` |
| 100 | `40458494` | `<rollout_summary>` |
| 101 | `40459458` | `<runtime>` |
| 102 | `40459972` | `<language>` |
| 103 | `40460081` | `<format>` |
| 104 | `40460391` | `<im_scene>` |
| 105 | `40460501` | `<agent_loop>` |
| 106 | `40460514` | `<tool_call_style>` |
| 107 | `40461826` | `<agent_loop>` |
| 108 | `40461906` | `<memory_system>` |
| 109 | `40461927` | `<memory_index>` |
| 110 | `40461946` | `<memory_recall_rules>` |
| 111 | `40462384` | `<tool_use>` |
| 112 | `40462538` | `<system_reminder>` |
| 113 | `40462587` | `<tool_call_style>` |
| 114 | `40462983` | `<lingxi_behavior>` |
| 115 | `40463861` | `<lingxi_reminders>` |
| 116 | `40463944` | `<system_reminder>` |
| 117 | `40464020` | `<error_handling>` |
| 118 | `40464491` | `<safety>` |
| 119 | `40464790` | `<delete_file_rules>` |
| 120 | `40465281` | `<ask_user_question_tool>` |
| 121 | `40465772` | `<render_widget_tool>` |
| 122 | `40465877` | `<search_strategy>` |
| 123 | `40466254` | `<timer_task_restriction>` |
| 124 | `40466392` | `<cloud_environment_restriction>` |
| 125 | `40466601` | `<file_handling_rules>` |
| 126 | `40467563` | `<skill_management_rules>` |
| 127 | `40468617` | `<sharing_files>` |
| 128 | `40472378` | `<response_ending>` |
| 129 | `40472855` | `<disclosure_prohibition>` |
| 130 | `40473338` | `<user_profile>` |

## 段频次（该区间内）

| 段 | 次数 |
|---|---|
| `<sharing_files>` | 12 |
| `<agent_loop>` | 8 |
| `<tool_call_style>` | 8 |
| `<system_reminder>` | 8 |
| `<noop>` | 6 |
| `<runtime>` | 4 |
| `<language>` | 4 |
| `<format>` | 4 |
| `<im_scene>` | 4 |
| `<memory_system>` | 4 |
| `<memory_index>` | 4 |
| `<memory_recall_rules>` | 4 |
| `<tool_use>` | 4 |
| `<error_handling>` | 4 |
| `<search_strategy>` | 4 |
| `<timer_task_restriction>` | 4 |
| `<cloud_environment_restriction>` | 4 |
| `<safety>` | 4 |
| `<delete_file_rules>` | 4 |
| `<ask_user_question_tool>` | 4 |
| `<render_widget_tool>` | 4 |
| `<file_handling_rules>` | 4 |
| `<skill_management_rules>` | 4 |
| `<response_ending>` | 4 |
| `<disclosure_prohibition>` | 4 |
| `<user_profile>` | 4 |
| `<group_chat>` | 2 |
| `<skills>` | 2 |
| `<assistant_behavior>` | 2 |
| `<assistant_reminders>` | 2 |
| `<lingxi_behavior>` | 2 |
| `<lingxi_reminders>` | 2 |
| `<rollout_summary>` | 2 |
| `<web_artifact>` | 1 |