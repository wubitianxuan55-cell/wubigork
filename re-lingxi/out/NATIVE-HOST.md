# lingxi-native-host.exe（静态盘点）

- 二进制: `C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi-native-host.exe`（6.7 MB，**Go**，module `…/cmd/native-host`）
- 包: `internal/nativehost`；符号 314 个 / 40 包

## 生命周期符号

- `nativehost.CDPError`
- `nativehost.CDPRequest`
- `nativehost.CDPResponse`
- `nativehost.EmbeddedServer`
- `nativehost.KillStaleNativeHost`
- `nativehost.ListenWithCleanup`
- `nativehost.LogDir`
- `nativehost.NMInbound`
- `nativehost.NMOutbound`
- `nativehost.NewServer`
- `nativehost.NewStdinIdleMonitor`
- `nativehost.PortCleanupResult`
- `nativehost.PortFilePath`
- `nativehost.ReadNM`
- `nativehost.Server`
- `nativehost.StartServer`
- `nativehost.StatusResponse`
- `nativehost.StdinIdleMonitor`
- `nativehost.UnmarshalNM`
- `nativehost.WriteNM`
- `nativehost.failEntry`
- `nativehost.isNativeHostBinary`
- `nativehost.killProcess`
- `nativehost.marshalNM`
- `nativehost.pendingEntry`
- `nativehost.portread`
- `nativehost.queryPortPids`
- `nativehost.queryProcessExePath`
- `nativehost.userHomeDir`
- `nativehost.withBodyLimit`
- `nativehost.writePortFile`

## 消息面（原文）

- `[NH-MSG] session ready: %s using`
- `[NH-HB] ping send failed: %v`
- `[NH-HB] ping marshal failed: %vreflect: Len of non-array type`
- `[NH-MSG] Plugin status: state=%ssync: Unlock of unlocked RWMutexsync: negative WaitGroup counter`
- `[NH-MSG] Unknown message type: %sreflect: slice index out of rangesync: RUnlock of unlocked RWMutex of method on nil interface valuereflect:`
- `[NH-CDP] encode error response: %vreflect: Field`
- `[NH-CDP] response dispatched: id=%dreflect: Fields`
- `[NH-HTTP] encode status response: %v`
- `[NH-CDP] encode success response: %v`
- `[NH-CDP] session_end send failed: %vreflect: NumField`
- `[NH-CDP] rejected: plugin not connected`
- `[NH-CDP] session_end marshal failed: %v`
- `[NH-MSG] cdp.response: id=%v session=%sreflect: OverflowUint of non-uint type`
- `[NH-MSG] cdp.event: method=%s session=%s`
- `[NH-CDP] encode disconnected response: %v`
- `[NH-HTTP] encode session_end response: %vcan't call pointer on a non-pointer Value`
- `[NH-CDP] encode ended-session response: %vsync/atomic: store`
- `[NH-CDP] encode plugin timeout response: %vreflect:`
- `[NH-MSG] Plugin connected (status: accepted)`
- `[NH-CDP] cdp.command write failed: id=%d err=%v`
- `[NH-CDP] cdp.command marshal failed: id=%d err=%v`
- `[NH-CDP] encode session-ready timeout response: %v`
- `[NH-CDP] waiting for cdp.response: id=%d method=%s`
- `[NH-MSG] session ready (legacy, no session_id): %sgo package net: dynamic selection of DNS resolver got CONTINUATION for stream %d; expected`
- `[NH-CDP] session_end: session=%s group=%s reason=%s`
- `[NH-CDP] got cdp.response: id=%d method=%s elapsed=%vreflect: non-interface type passed to Type.Implementssmall map with no empty slot (conc`
- `[NH-MSG] session_end from plugin: session=%s reason=%s`
- `[NH-CDP] response for unknown id: %d (pending_count=%d)reflect:`
- `[NH-CDP] rejected: session ended by plugin: %s reason=%s`
- `[NH-CDP] caller disconnected: id=%d method=%s session=%s`
- `[NH-CDP] rejected unhealthy session: session=%s reason=%qsync: WaitGroup misuse: Add called concurrently with Waitsync: WaitGroup.Add called`
- `[NH-HB] plugin reconnected (pong received after disconnect)sync: WaitGroup is reused before previous Wait has returnedsync/atomic: store of`
- `[NH-MSG] Plugin error: code=%s message=%s session=%s fatal=%vexec: Cmd started a Process but leaked without a call to Waittls: unsupported c`
- `[NH-CDP] sending cdp.command: id=%d method=%s session=%s brief=%shttp: response.WriteHeader on hijacked connection from %s (%s:%d)tls:`
- `[NH-HB] plugin unresponsive (last pong %v ago), marking disconnectedreflect:`
- `[NH-CDP] TIMEOUT waiting for cdp.response: id=%d method=%s elapsed=%v`
- `[NH-CDP] command error: id=%d method=%s session=%s code=%d message=%q%s matches more methods than %s, but has a more specific path pattern%s`
- `[NH-CDP] TIMEOUT waiting for session_ready: id=%d method=%s session=%s elapsed=%v`

## 错误/清理串（原文）

- `native-host: listen: %w`
- `native-host server: %vinvalid boolean flag %s: %vflag needs an argument: -%sinput does not match formatinconsistent proces`
- `native-host: http shutdown: %wreflect: Elem of invalid type reflect: Len of non-array type`
- `native-host: close listener: %w[NH-HB] ping marshal failed: %vreflect: Len of non-array type`
- `native-host: remove port file: %wresponse timeout: id=%d method=%s[NH-MSG] Unknown message type: %sreflect: slice index ou`
- `native-host log writer: %v flag provided but not defined: -%s`
- `native-host: write port file %s: %w[native-host] port %d written to %s[native-host] HTTP server error: %v[native-host] sta`
- `native-host artifact version and exitNative host started, reading from plugin.../memory/classes/metadata/mcache/inuse:byte`
- `native-host: already running in this process[native-host] stdin closed or read error: %v[NH-MSG] Plugin connected (status:`
- `native-host: timeout waiting for session readyreflect:`
- `native-host pid %d (%s)[native-host] WARN: failed to send session_start: %vstdin idle for %v, exiting to prevent zombie pr`
- `native-host started, log dir: %s (daily rotation, gzip, retain %d days) ===`
- `native-host: timeout waiting for plugin response: id=%d method=%s; underlying command may still be runningasn1: time did n`

## 结构读法

1. 起一个 **回环 HTTP 服务**（`ListenWithCleanup`/`StartServer`），把端口写到 **端口文件**（`PortFilePath`/`writePortFile`），
   并用 `netstat -ano | findstr :<port>` + `queryPortPids`/`killProcess` 清掉上一轮残留（`KillStaleNativeHost`）。
2. 浏览器侧「插件」连上来后走 `[NH-MSG]` 消息：`session_start` / `session_end` / `cdp.event` / `cdp.response`；
   未连插件时 CDP 请求被拒：`[NH-CDP] rejected: plugin not connected`。
3. 消息编解码是 `marshalNM`/`UnmarshalNM`/`ReadNM`/`WriteNM`，带 `withBodyLimit` 的体长上限；`Content-Length` 出现 11 次 → HTTP 体分帧。
4. 心跳/防挂死：`StdinIdleMonitor`（stdin 空闲监控）+ `response timeout: id=%d method=%s`。

→ 对 gaea 的价值：这是「**本地 HTTP + 插件长连 + CDP 转发**」的完整最小实现，比直接 embed 浏览器简单，且天然跨进程。