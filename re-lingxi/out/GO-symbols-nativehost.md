# Go 符号盘点 · `lingxi-native-host.exe`

- 二进制: `C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi-native-host.exe`
- 命中符号 **314** 个，**40** 个包（静态串扫描；不含类型/未导出内联信息）

## 包 × 符号数（前 60）

| 包 | 符号数 |
|---|---|
| `vendor/golang.org/x/text/unicode` | 45 |
| `ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/internal` | 41 |
| `vendor/golang.org/x/net/http` | 26 |
| `vendor/golang.org/x/net/http2` | 25 |
| `vendor/golang.org/x/net/dns` | 22 |
| `vendor/golang.org/x/crypto` | 21 |
| `vendor/golang.org/x/net` | 20 |
| `golang.org/x/sys` | 18 |
| `vendor/golang.org/x/sys` | 8 |
| `Files/Go/src/vendor/golang.org/x/text/unicode/norm` | 8 |
| `vendor/golang.org/x/crypto/internal` | 7 |
| `github.com/sirupsen` | 7 |
| `Files/Go/src/vendor/golang.org/x/net/idna` | 6 |
| `.eq.golang.org/x/sys` | 5 |
| `Files/Go/src/vendor/golang.org/x/net/http2/hpack` | 5 |
| `Files/Go/src/vendor/golang.org/x/crypto/chacha20poly1305` | 4 |
| `ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/core` | 4 |
| `vendor/golang.org/x/text/secure` | 3 |
| `Files/Go/src/vendor/golang.org/x/crypto/internal/poly1305` | 3 |
| `.eq.ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/internal` | 3 |
| `Files/Go/src/vendor/golang.org/x/crypto/chacha20` | 3 |
| `Files/Go/src/vendor/golang.org/x/crypto/cryptobyte` | 3 |
| `Files/Go/src/vendor/golang.org/x/sys/cpu` | 3 |
| `Files/Go/src/vendor/golang.org/x/text/unicode/bidi` | 2 |
| `Files/Go/src/vendor/golang.org/x/net/dns/dnsmessage` | 2 |
| `.eq.vendor/golang.org/x/net/http` | 2 |
| `Files/Go/src/vendor/golang.org/x/text/secure/bidirule` | 2 |
| `Files/Go/src/vendor/golang.org/x/net/http/httpguts` | 2 |
| `.eq.vendor/golang.org/x/net/http2` | 2 |
| `/Users/Administrator/go/pkg/mod` | 2 |
| `.hash.vendor/golang.org/x/net/http2` | 1 |
| `.eq.vendor/golang.org/x/net/dns` | 1 |
| `Files/Go/src/vendor/golang.org/x/net/http/httpproxy` | 1 |
| `Files/Go/src/vendor` | 1 |
| `.eq.github.com/sirupsen` | 1 |
| `.eq.ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/core` | 1 |
| `Files/Go/src/vendor/golang.org/x/crypto/internal/alias` | 1 |
| `vendor` | 1 |
| `.eq.vendor/golang.org/x/net` | 1 |
| `.eq.vendor/golang.org/x/sys` | 1 |

## 各包函数清单（按包名排序，包内符号数 ≥2 且属 lingxi/ksogit 与本仓相关的先列）

### `ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/internal`

- `buildinfo.Current`
- `buildinfo.Info.String`
- `diagnosticlog.SafeText`
- `diagnosticlog.init`
- `diagnosticlog.stripURLQuery`
- `diagnosticlog.truncate`
- `nativehost.KillStaleNativeHost`
- `nativehost.KillStaleNativeHost.Printf.func1`
- `nativehost.KillStaleNativeHost.Printf.func2`
- `nativehost.KillStaleNativeHost.Printf.func3`
- `nativehost.KillStaleNativeHost.Printf.func4`
- `nativehost.KillStaleNativeHost.Printf.func5`
- `nativehost.ListenWithCleanup`
- `nativehost.ListenWithCleanup.Printf.func1`
- `nativehost.ListenWithCleanup.Printf.func2`
- `nativehost.ListenWithCleanup.Printf.func3`
- `nativehost.LogDir`
- `nativehost.NewServer`
- `nativehost.NewStdinIdleMonitor`
- `nativehost.PortFilePath`
- `nativehost.ReadNM`
- `nativehost.StartServer`
- `nativehost.StartServer.Printf.func3`
- `nativehost.StartServer.Printf.func4`
- `nativehost.StartServer.func1`
- `nativehost.StartServer.func1.Printf`
- `nativehost.StartServer.func1.deferwrap1`
- `nativehost.StartServer.func2`
- `nativehost.StartServer.func2.deferwrap1`
- `nativehost.StartServer.gowrap1`
- `nativehost.StartServer.gowrap2`
- `nativehost.UnmarshalNM`
- `nativehost.WriteNM`
- `nativehost.isNativeHostBinary`
- `nativehost.killProcess`
- `nativehost.marshalNM`
- `nativehost.queryPortPids`
- `nativehost.queryProcessExePath`
- `nativehost.userHomeDir`
- `nativehost.withBodyLimit`
- `nativehost.writePortFile`

### `ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/core`

- `applog.NewRotateWriter`
- `applog.NewRotateWriter.gowrap1`
- `applog.compressFile`
- `applog.compressFile.func1`

### `.eq.ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/internal`

- `nativehost.CDPError`
- `nativehost.StdinIdleMonitor`
- `nativehost.pendingEntry`

### `.eq.ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/core`

- `applog.RotateWriter`
