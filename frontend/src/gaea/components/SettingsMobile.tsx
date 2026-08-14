import { useState, useEffect, useCallback } from "react";
import { Smartphone, QrCode, RefreshCw, Copy, Check, ShieldAlert } from "../icons";

/** SettingsMobile — 移动端访问配置卡片（P4-1 模块收口：移动端已冻结，仅保留说明） */
export function MobileSection() {
  const [lanEnabled, setLanEnabled] = useState(false);
  const [lanURL, setLanURL] = useState("");
  const [qrURL, setQrURL] = useState("");
  const [token, setToken] = useState("");
  const [tokenVisible, setTokenVisible] = useState(false);
  const [copied, setCopied] = useState(false);

  // 检测 LAN URL
  useEffect(() => {
    const { hostname, port } = window.location;
    if (hostname === "127.0.0.1" || hostname === "localhost") {
      // 尝试获取本机 LAN IP（通过 WebRTC 或预设）
      setLanURL(`http://${getLocalIP()}:${port || "8787"}`);
    } else {
      setLanURL(window.location.origin);
      setLanEnabled(true);
    }
  }, []);

  const handleToggle = useCallback(() => {
    setLanEnabled((v) => !v);
    // 实际实现会调用 app.SetServeConfig(...) 来启用 LAN 绑定
    // 这里只展示 UI
  }, []);

  const handleGenerateToken = useCallback(() => {
    const t = Array.from({ length: 32 }, () =>
      "abcdefghijklmnopqrstuvwxyz0123456789".charAt(Math.floor(Math.random() * 36))
    ).join("");
    setToken(t);
  }, []);

  const handleCopy = useCallback(() => {
    if (token) {
      navigator.clipboard.writeText(token).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    }
  }, [token]);

  // QR code 使用 Google Chart API（无额外依赖）
  const qrSrc = qrURL
    ? `https://chart.googleapis.com/chart?cht=qr&chs=256x256&chl=${encodeURIComponent(qrURL)}`
    : null;

  return (
    <div className="flex flex-col gap-3">
      {/* 启用 LAN 访问 */}
      <div className="flex items-center justify-between px-3 py-2 rounded-lg border border-border-soft bg-bg-soft">
        <div className="flex items-center gap-2">
          <Smartphone size={16} className="text-fg-dim" />
          <span className="text-sm text-fg">移动端访问</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded-full border border-warn/40 text-warn">已冻结</span>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={lanEnabled}
            onChange={handleToggle}
          />
          <div className="w-9 h-5 bg-border-soft rounded-full peer peer-checked:bg-accent after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4" />
        </label>
      </div>

      {lanEnabled && (
        <>
          {/* LAN URL + QR */}
          <div className="flex flex-col items-center gap-3 p-4 rounded-lg border border-border-soft bg-bg-soft">
            <div className="flex items-center gap-2 text-sm">
              <span className="w-2 h-2 rounded-full bg-ok animate-pulse" />
              <span className="text-fg-dim">局域网可访问</span>
            </div>

            {/* QR 码 */}
            {qrSrc ? (
              <img
                src={qrSrc}
                alt="QR Code"
                className="w-48 h-48 rounded-lg border border-border-soft bg-white p-2"
              />
            ) : (
              <div
                className="w-48 h-48 rounded-lg border border-border-soft bg-bg flex items-center justify-center text-fg-faint text-xs cursor-pointer hover:bg-bg-soft"
                onClick={() => setQrURL(lanURL)}
              >
                <QrCode size={32} className="mb-1 opacity-40" />
                <span>点击生成二维码</span>
              </div>
            )}

            {/* LAN URL */}
            <div className="w-full flex items-center gap-2 px-3 py-2 bg-bg rounded font-mono text-xs text-accent truncate">
              <span className="shrink-0 text-fg-faint">URL:</span>
              <span className="truncate">{lanURL}</span>
              <button
                className="shrink-0 border-0 bg-transparent text-fg-faint hover:text-fg cursor-pointer p-1"
                onClick={() => navigator.clipboard.writeText(lanURL)}
                title="复制 URL"
              >
                <Copy size={12} />
              </button>
            </div>
          </div>

          {/* Token 鉴权 */}
          <div className="flex flex-col gap-2 p-3 rounded-lg border border-border-soft bg-bg-soft">
            <div className="flex items-center gap-2 text-sm">
              <ShieldAlert size={14} className="text-warning" />
              <span className="text-fg">Bearer Token 鉴权</span>
              <span className="text-fg-faint text-xs">（可选）</span>
            </div>

            {!token ? (
              <button
                className="flex items-center gap-1.5 px-3 py-1.5 border border-accent/30 rounded-md bg-transparent text-accent text-xs cursor-pointer hover:bg-accent/10 self-start"
                onClick={handleGenerateToken}
              >
                <RefreshCw size={12} />
                生成 Token
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <code className="flex-1 px-2 py-1.5 bg-bg rounded font-mono text-xs text-fg-dim truncate">
                  {tokenVisible ? token : `${token.slice(0, 8)}...${token.slice(-4)}`}
                </code>
                <button
                  className="border-0 bg-transparent text-fg-faint hover:text-fg cursor-pointer p-1"
                  onClick={() => setTokenVisible((v) => !v)}
                  title={tokenVisible ? "隐藏" : "显示"}
                >
                  {tokenVisible ? "🙈" : "👁"}
                </button>
                <button
                  className="border-0 bg-transparent text-fg-faint hover:text-fg cursor-pointer p-1"
                  onClick={handleCopy}
                  title="复制"
                >
                  {copied ? <Check size={14} className="text-ok" /> : <Copy size={14} />}
                </button>
              </div>
            )}

            <p className="text-[10px] text-fg-faint leading-relaxed">
              设置 Token 后所有 HTTP 请求需携带 <code className="bg-bg px-1 rounded">Authorization: Bearer &lt;token&gt;</code> 头。
              手机扫码后会自动带上 Token。
            </p>
          </div>

          {/* 使用说明 */}
          <div className="px-3 py-2 text-xs text-fg-faint leading-relaxed space-y-1">
            <p className="font-medium text-fg-dim">📱 使用方式：</p>
            <ol className="list-decimal list-inside space-y-0.5">
              <li>手机扫码打开 gaea</li>
              <li>浏览器底部菜单 → "添加到主屏幕"</li>
              <li>桌面出现 gaea 图标，全屏 PWA 体验</li>
              <li>从手机发送消息，桌面 agent 处理并回复</li>
            </ol>
          </div>
        </>
      )}
    </div>
  );
}

/** 获取本机局域网 IP（简化版） */
function getLocalIP(): string {
  try {
    // 通过 WebRTC 获取
    const pc = new RTCPeerConnection({ iceServers: [] });
    pc.createDataChannel("");
    return new Promise<string>((resolve) => {
      pc.onicecandidate = (e) => {
        if (e.candidate) {
          const match = /([0-9]{1,3}\.){3}[0-9]{1,3}/.exec(e.candidate.candidate);
          if (match && !match[0].startsWith("127.") && !match[0].startsWith("192.168.") && !match[0].startsWith("10.")) {
            // Prefer non-private IPs first, but LAN is fine
          }
          if (match) {
            const ip = match[0];
            if (ip.startsWith("192.168.") || ip.startsWith("10.") || ip.startsWith("172.")) {
              resolve(ip);
              pc.close();
            }
          }
        }
      };
      setTimeout(() => {
        resolve("192.168.1.100"); // fallback
        pc.close();
      }, 1000);
    }) as unknown as string;
  } catch {
    return "192.168.1.100";
  }
}
