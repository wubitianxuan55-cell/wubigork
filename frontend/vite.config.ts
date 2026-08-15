import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Wails 内嵌资产服务器不支持 CORS 头，浏览器会拒绝加载带 crossorigin 的 module script。
// 同源的 ESM 脚本无需 CORS，去掉 crossorigin 即可正常加载。
// 同时注入错误捕获脚本（延迟到 DOMContentLoaded 再渲染错误 UI）。
function wailsCompatPlugin() {
  return {
    name: 'wails-compat',
    transformIndexHtml(html: string) {
      return html
        .replace(/\s*crossorigin\b/gi, '')
        // 注入 JS 错误捕获脚本（生产环境诊断用）
        .replace(
          '<head>',
          `<head>
<script>
(function(){
  var errors=[];
  window.onerror=function(m,u,l,c,e){errors.push({msg:m,line:l,col:c,stack:e&&e.stack||''});};
  window.addEventListener('unhandledrejection',function(ev){errors.push({msg:'[Promise] '+(ev.reason&&ev.reason.message||ev.reason),stack:ev.reason&&ev.reason.stack||''});});
  document.addEventListener('DOMContentLoaded',function(){
    if(!errors.length)return;
    var d=document.createElement('div');
    d.style.cssText='position:fixed;top:0;left:0;right:0;z-index:99999;background:#1a0000;color:#ff4444;font:12px monospace;padding:12px;white-space:pre-wrap;max-height:50vh;overflow:auto;border-bottom:2px solid red';
    d.textContent=errors.map(function(e){return e.msg+(e.line?' (line '+e.line+':'+e.col+')':'')+'\\n'+e.stack;}).join('\\n\\n');
    document.body.insertBefore(d,document.body.firstChild);
  });
})();
</script>`)
    },
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss(), wailsCompatPlugin()],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
  },
  server: {
    port: 5173,
    strictPort: true,
    host: '0.0.0.0', // 允许手机访问 Vite dev server
    // 网页版对齐桌面端：/api 转发到 Go 内核的 HTTP 调试桥接
    // （由 GAEA_HTTP_PORT 启动，默认 8080；本地端口被占用时可用
    //   GAEA_PROXY_PORT 指向实际桥接端口）。桥接未启动时这些请求会失败，
    // 前端页面会退化为空数据而不是渲染崩溃。
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${process.env.GAEA_PROXY_PORT || '8080'}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
})
