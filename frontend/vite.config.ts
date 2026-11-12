import { defineConfig, defaultExclude } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import os from 'node:os'
import { LOGIC_ENV_TEST_FILES } from './src/test/logicEnvFiles'

// worker 上限跟机器走，避免把配置钉死在本机：
// 逻辑核折半 ≈ 物理核（超线程不提供真吞吐），再夹到 [4, 8]。
// 本机 32 逻辑核 → 8；GitHub 4 vCPU runner → 4；8 逻辑核笔记本 → 4。
const logical = typeof os.availableParallelism === 'function' ? os.availableParallelism() : os.cpus().length
const TEST_MAX_WORKERS = Math.max(4, Math.min(8, Math.floor(logical / 2)))

// 双项目共享的预算（显式写进每个项目，vitest 4 的 projects 对根配置的继承
// 语义存在版本差异，不依赖继承）。
const TEST_COMMON = {
  css: true,
  maxWorkers: TEST_MAX_WORKERS,
  testTimeout: 30000,
  hookTimeout: 20000,
}

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
    // 双项目拆环境（2026-11-12 成本刀）：全量 4301 用例的实际执行只占 CPU
    // 总账 ~12.5%（544s），其余 ~87.5% 是每文件固定开销——模块 import
    // 2478s（57%）+ jsdom 环境 1098s（25%）+ setup 212s。纯逻辑测试文件
    // （无 React / 无 DOM 直接引用，清单见 src/test/logicEnvFiles.ts）走
    // node 环境免掉 jsdom 环境成本；组件测试仍走 jsdom。
    // 两项目 include/exclude 由同一份清单构造上互斥：vitest 4 对同时命中
    // 多个项目的文件会各跑一遍（globTestSpecifications 无跨项目去重），
    // 不能依赖匹配顺序。
    // 门禁可信化（2026-09-10 实测收口，取代「假红就复跑」）：
    // 每个测试文件都要付一遍固定成本——jsdom 环境 ~0.65s + 重模块图导入
    // ~2.1s（FilePreviewModal 单文件实测 5.4s；324 文件累计 import 1940s /
    // environment 1300s）。本机 16 物理核 / 32 逻辑核，vitest 默认按逻辑核
    // 开 ~31 个 worker：超配后单 worker 墙钟可达 CPU 时间的数倍，重组件
    // （ContextView / TrajectoryView / MemoryPanel / FilePreviewModal）首个
    // await import 被计入 testTimeout → 15~20s 超时假红；超时用例迟到的
    // render 还会污染下一个用例（Found multiple elements ...）。
    // 上限压到物理核的一半：墙钟 ≈ CPU 时间，单用例延迟有余量。
    // 预算是给「worker 短暂抢不到物理核」留的余量，不是用来兜住真死锁：
    // 真回归是断言失败（立即出现），不受此影响；超时若在隔离复跑下仍复现，
    // 那就是真缺陷。
    // 公共预算在两项目各自显式声明（TEST_COMMON），不依赖根配置继承语义。
    projects: [
      {
        test: {
          name: 'logic',
          environment: 'node',
          setupFiles: [],
          include: LOGIC_ENV_TEST_FILES,
          ...TEST_COMMON,
        },
      },
      {
        test: {
          name: 'ui',
          environment: 'jsdom',
          // isolate:false 已两轮证伪（2026-11-12）：①一刀全开=126s 但 51 文件/156
          // 用例跨文件泄漏假红；②渐进迁移（快考场+实证清单分批搬）三轮流产后仍
          // 每轮爆 10~18 个新红——非隔离下「跑绿名单」不可信，换邻居/换顺序就翻脸，
          // 本机 ceiling 实测 ~170s（非 126s）。要吃这 ~2.5x 须先做全仓测试卫生
          // 审计（每文件自带清理/mock 还原，按 worker 复用模块态设计），不是配置刀。
          // 三考场统一池的原因：vitest 4 对 maxWorkers/pool 不同的项目强制拆
          // sequence 组、组间串行。
          setupFiles: './src/test/setup.ts',
          include: ['src/**/*.test.{ts,tsx}'],
          exclude: [...defaultExclude, ...LOGIC_ENV_TEST_FILES],
          ...TEST_COMMON,
        },
      },
    ],
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
