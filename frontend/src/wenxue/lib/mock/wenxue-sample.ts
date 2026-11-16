// ?mock=wenxue 场景样例工作空间（雾港来信）：章节/定稿/角色/设定/时间线
// 五目录真实形态样例——mock.ts 场景注释承诺的兑现（2026-11-17 审查批3 刀C，
// 此前 ?mock=wenxue 下四个 Hub 恒空态）。仅供浏览器开发/走查使用，纯前端
// 数据不触及 Go 内核。

export const WENXUE_SAMPLE_BOOK = "C:/AI/tuwen/雾港来信";

const BOOK_INFO = `# 书籍信息

> 本文件由新建小说流程自动生成，会话启动时自动注入；可直接编辑，改动随下次会话生效。

- 书名：雾港来信
- 题材：悬疑·海洋奇谈
- 目标字数：60 万
- 成人等级：全年龄

## 简介

守夜人沈青梧回到阔别十年的雾港，一封没有署名的灯塔来信，把三十年前的沉船名单重新摊开在她面前。

## 内容分级说明

- 全年龄：无成人内容；情感与冲突克制处理，亲密与暴力均不直接呈现。
- 唯一恒守线：涉未成年角色的性内容一律禁止。
`;

const CHAPTER_1 = `# 第一章 雾港夜航

雾在傍晚准时漫进港口，像有人在城外拧开了一台巨大的白色机器。

沈青梧把行李箱的拉杆压到底，抬头望向码头尽头的灯塔——三十年了，它还站在那里，光却比记忆里暗得多。舷梯放下的时候，她听见身后有人低声说：又来一个想上灯塔的。
`;

const CHAPTER_2 = `# 第二章 灯塔来信

信是塞在门缝里的，没有邮票，也没有署名。

信纸只有一页，字迹被海水洇过：『名单上的第七个名字，还活着。』沈青梧把信纸翻过来，背面用铅笔描着一张吃水线极深的货船侧影——雾航号，三十年前沉在灯火礁的那一艘。
`;

const FINAL_1 = `# 第一章 雾港夜航

雾在傍晚准时漫进港口，像有人在城外拧开了一台巨大的白色机器。

沈青梧把行李箱的拉杆压到底，抬头望向码头尽头的灯塔——三十年了，它还站在那里，光却比记忆里暗得多。（定稿）
`;

const CHAR_SHEN = `# 沈青梧

## 设定

灯塔守夜人，前海事档案员；三十年前沉船事故遗属，回港追查真相。

## 外貌

二十八岁，短发利落，左手腕有一道旧疤；常年穿一件洗旧的藏青色外套。

## 性格

外冷内热，观察力极强，习惯用记录代替倾诉；对「名单」两个字异常敏感。

## 口头禅

「先记下来，再下判断。」

## 关系

陆知远：旧识，立场可疑的信件投递人（存疑）。

## 当前状态

已接下守夜人一职，住在灯塔附楼；正在整理三十年前的港口档案。
`;

const CHAR_LU = `# 陆知远

## 设定

雾港渔行账房，沉船名单的知情者之一；既想保住秘密，又忍不住递出线索。

## 外貌

四十岁上下，微胖，指节粗大；总戴一副只有半边镜腿的老花镜。

## 性格

圆滑胆小，但底线未泯；酒后话多，醒后翻供。

## 口头禅

「有些账，记在心里就行。」

## 关系

沈青梧：看着她长大的长辈，也是当年名单的抄写人。

## 当前状态

频繁出入码头酒馆，暗中观察沈青梧查到了哪一步。
`;

const WORLDVIEW = `# 雾港世界观

雾港是南方海岸线上最后一座手工灯塔城市，港口经济依赖渔业与船舶维修。三十年前的雾航号沉船事件是全城默契回避的禁忌：官方结论是锅炉爆炸，但渔民中流传着「名单」的说法——船上不该载货的货，与不该上船的人。

灯塔的光有规律地暗下去，被本地人称作「灯语」，是守夜人之间传讯的旧法。
`;

const POWER = `# 灯语（力量体系）

灯语不是魔法，而是一套靠灯光明灭节奏传递信息的编码：长亮=平安，两短一长=有人到港，连续急闪=封港。守夜人家族代代口传，掌握完整灯语表的人如今只剩灯塔顶层的黑匣子。

本作不设超自然能力；一切「异象」最终都要能回到人为解释。
`;

const TIMELINE = `# 时间线

- 第1章 · 沈青梧抵达雾港，接下灯塔守夜人一职。
- 第2章 · 一封没有署名的来信揭开三十年前沉船名单的一角。
`;

/** 样例文件表：相对路径（posix）→ 文件内容。 */
export const wenxueSampleFiles: Record<string, string> = {
  "书籍信息.md": BOOK_INFO,
  "章节/001-雾港夜航.md": CHAPTER_1,
  "章节/002-灯塔来信.md": CHAPTER_2,
  "定稿/001-雾港夜航.md": FINAL_1,
  "角色/沈青梧.md": CHAR_SHEN,
  "角色/陆知远.md": CHAR_LU,
  "设定/世界观-雾港.md": WORLDVIEW,
  "设定/力量体系-灯语.md": POWER,
  "时间线/时间线.md": TIMELINE,
};

export interface WenxueSampleEntry {
  name: string;
  isDir: boolean;
  size: number;
}

/** 按样例文件表服务目录列举（与 Go ListDir 形状一致）；非样例路径返回 undefined=回落。 */
export function wenxueSampleListDir(rel: string): WenxueSampleEntry[] | undefined {
  const norm = rel.replace(/\\/g, "/").replace(/^\.\//, "").replace(/\/+$/, "");
  const prefix = norm === "" || norm === "." ? "" : norm + "/";
  const out = new Map<string, WenxueSampleEntry>();
  let hit = false;
  for (const p of Object.keys(wenxueSampleFiles)) {
    if (prefix !== "" && !p.startsWith(prefix)) continue;
    if (prefix === "" && !p.includes("/")) {
      hit = true;
      out.set(p, { name: p, isDir: false, size: wenxueSampleFiles[p].length });
      continue;
    }
    if (prefix === "") continue;
    hit = true;
    const tail = p.slice(prefix.length);
    const slash = tail.indexOf("/");
    if (slash < 0) out.set(tail, { name: tail, isDir: false, size: wenxueSampleFiles[p].length });
    else out.set(tail.slice(0, slash), { name: tail.slice(0, slash), isDir: true, size: 0 });
  }
  return hit ? [...out.values()] : undefined;
}

/** 按样例文件表取正文；rel 支持 ./ 与反斜杠形态。 */
export function wenxueSampleFile(rel: string): string | undefined {
  const norm = rel.replace(/\\/g, "/").replace(/^\.\//, "");
  return wenxueSampleFiles[norm];
}
