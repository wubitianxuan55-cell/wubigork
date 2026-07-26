import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

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
  plugins: [react(), wailsCompatPlugin()],
  server: {
    port: 5173,
    strictPort: true,
    host: '0.0.0.0', // 允许手机访问 Vite dev server
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
})
