// AppBindings mirrors desktop/app.go's exported method set. Keep in sync by hand
// (or regenerate with `wails generate module` and import wailsjs instead).
//
// Compile-time drift check: when a Go method is added/renamed but AppBindings is
// not updated, the type assertion below catches it at build time.  Fix: add the
// missing method to AppBindings, then run `pnpm typecheck`.

// P3 结构版1：AppBindings 按 Go 门面域拆为 10 个分域接口（core/office/memory/
// cost/model/voice/chat/novel/image/charlib）再组合——方法名/签名/types 逐一同，
// 保持 interface 形态（extends 组合，成员冲突会编译报错兜底）。
import type { CoreBindings } from "./core";
import type { OfficeBindings } from "./office";
import type { MemoryBindings } from "./memory";
import type { CostBindings } from "./cost";
import type { ModelBindings } from "./model";
import type { VoiceBindings } from "./voice";
import type { ChatBindings } from "./chat";
import type { NovelBindings } from "./novel";
import type { ImageBindings } from "./image";
import type { CharLibBindings } from "./charlib";
import type { SinBindings } from "./sin";

export interface AppBindings
  extends CoreBindings,
    OfficeBindings,
    MemoryBindings,
    CostBindings,
    ModelBindings,
    VoiceBindings,
    ChatBindings,
    NovelBindings,
    ImageBindings,
    CharLibBindings,
    SinBindings {}
