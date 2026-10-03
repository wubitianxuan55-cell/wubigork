package browser

// manager_evaluate.go — Runtime.evaluate 封装与页面 JS 片段（批 26 GA5-08
// 文件拆分，自 manager.go 原位搬移，零逻辑改动）：evaluate/evaluateIn 主/子帧
// 执行 + okField/jsErr/jsString/boolJS 辅助 + jsMeta/jsRead/jsSnapshot/jsClick/
// jsType/jsScroll 六段页面 JS。调用方（页面操作）见 manager_actions.go。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ── Runtime.evaluate 封装与 JS 片段 ─────────────────────────────────────

// okField 每段页面 JS 返回值的公共头：JS 内 try/catch，失败带 error 文本。
type okField struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// evaluate 在 active tab 的默认执行上下文执行一段 JS 并把返回值解进 out
// （returnByValue + awaitPromise）。
func (m *Manager) evaluate(ctx context.Context, expression string, out any) error {
	return m.evaluateIn(ctx, expression, 0, out)
}

// evaluateIn 同 evaluate，但可指定 executionContextId（>0 = iframe 隔离世界；
// 0 = 主文档默认上下文）。
func (m *Manager) evaluateIn(ctx context.Context, expression string, contextID int, out any) error {
	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return errors.New("browser: 会话未建立")
	}
	var resp struct {
		Result struct {
			Type        string          `json:"type"`
			Value       json.RawMessage `json:"value"`
			Description string          `json:"description"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	params := map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	}
	if contextID > 0 {
		params["contextId"] = contextID
	}
	if err := conn.Call(ctx, "Runtime.evaluate", params, &resp); err != nil {
		return err
	}
	if resp.ExceptionDetails != nil {
		d := resp.ExceptionDetails.Exception.Description
		if d == "" {
			d = resp.ExceptionDetails.Text
		}
		return fmt.Errorf("browser: JS 执行失败: %s", d)
	}
	if out != nil && len(resp.Result.Value) > 0 {
		if err := json.Unmarshal(resp.Result.Value, out); err != nil {
			return fmt.Errorf("browser: JS 返回值解析失败: %w", err)
		}
	}
	return nil
}

// jsErr 把页面 JS 的 {ok:false,error} 转成 Go 错误（未找到元素 → 语义化哨兵）。
func jsErr(msg string) error {
	if strings.Contains(msg, "未找到元素") {
		return fmt.Errorf("%w: %s", ErrElementNotFound, msg)
	}
	return errors.New(msg)
}

// jsString 把 Go 字符串编码成 JS 字符串字面量（json.Marshal 转义即安全注入）。
func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// boolJS 布尔值注入。
func boolJS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// jsMeta 取标题与落点 URL（导航后调用）。__gaeaMeta 为 fake CDP 的匹配 token。
const jsMeta = `(function(){try{var t='__gaeaMeta';return {ok:true,title:document.title||"",url:location.href};}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsRead 读全文或 selector 局部文本（limit 参数 %d，selector 字面量 %s）。
const jsRead = `(function(){try{
var limit=%d; var sel=%s; var text;
if(sel){var el=document.querySelector(sel);if(!el)return {ok:false,error:"未找到元素："+sel};text=el.innerText||"";}
else{text=document.body?document.body.innerText:"";}
if(text.length>limit)text=text.slice(0,limit)+"…[已截断]";
return {ok:true,title:document.title||"",url:location.href,text:text};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsSnapshot 给可交互元素标 data-gaea-ref，登记 window.__gaeaRefs 与
// __gaeaEpoch（refs 代数，跨调用持久、导航后随页面重置）。
const jsSnapshot = `(function(){try{
var MAXREF=%d;
function cssPath(el){
 if(el.id)return '#'+el.id;
 var parts=[],cur=el,depth=0;
 while(cur&&cur.nodeType===1&&depth<4){
  depth++;
  if(cur.id){parts.unshift('#'+cur.id);break;}
  var seg=(cur.tagName||'').toLowerCase();
  var p=cur.parentNode;
  if(p&&p.children){
   var same=0,idx=0;
   for(var i=0;i<p.children.length;i++){
    if(p.children[i].tagName===cur.tagName){same++;if(p.children[i]===cur)idx=same;}
   }
   if(same>1&&idx>0)seg+=':nth-of-type('+idx+')';
  }
  parts.unshift(seg);
  cur=cur.parentNode;
 }
 return parts.join(' > ');
}
var nodes=document.querySelectorAll('a,button,input,textarea,select,[onclick],[role=button]');
var items=[];
for(var i=0;i<nodes.length&&items.length<MAXREF;i++){
 var el=nodes[i];
 if(el.disabled)continue;
 var r=el.getBoundingClientRect();
 if(r.width===0&&r.height===0)continue;
 var ref=items.length+1;
 el.setAttribute('data-gaea-ref',String(ref));
 var tag=(el.tagName||'').toLowerCase();
 var text=(tag==='input'||tag==='textarea')?(el.value||el.placeholder||''):(el.innerText||el.getAttribute('aria-label')||'');
 text=String(text).replace(/\s+/g,' ').trim().slice(0,80);
 items.push({ref:ref,tag:tag,text:text,path:cssPath(el)});
}
window.__gaeaRefs=items;
window.__gaeaEpoch=(window.__gaeaEpoch||0)+1;
return {ok:true,title:document.title||"",url:location.href,epoch:window.__gaeaEpoch,items:items};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsClick 按目标选择器点击（先滚到视野中央），返回元素文本确认。
const jsClick = `(function(){try{
var op='gaeaClick'; var target=%s;
var el=document.querySelector(target);
if(!el)return {ok:false,error:"未找到元素："+target};
el.scrollIntoView({block:'center'});
el.click();
var t=(el.innerText||el.value||'').trim().slice(0,120);
return {ok:true,text:t};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsType 聚焦并写入值：input/textarea 用原生 setter + input/change 事件
// （React 受控组件兼容）；submit 时提交所在表单或退化点击。
const jsType = `(function(){try{
var op='gaeaType'; var target=%s; var value=%s; var doSubmit=%s;
var el=document.querySelector(target);
if(!el)return {ok:false,error:"未找到元素："+target};
el.scrollIntoView({block:'center'});
el.focus();
var tag=(el.tagName||'').toLowerCase();
if(tag==='input'||tag==='textarea'){
 var proto=tag==='input'?window.HTMLInputElement.prototype:window.HTMLTextAreaElement.prototype;
 var desc=Object.getOwnPropertyDescriptor(proto,'value');
 if(desc&&desc.set){desc.set.call(el,value);}else{el.value=value;}
 el.dispatchEvent(new Event('input',{bubbles:true}));
 el.dispatchEvent(new Event('change',{bubbles:true}));
}else if(el.isContentEditable){
 el.textContent=value;
 el.dispatchEvent(new Event('input',{bubbles:true}));
}else{
 el.value=value;
 el.dispatchEvent(new Event('change',{bubbles:true}));
}
if(doSubmit){
 var f=el.form||el.closest('form');
 if(f){if(f.requestSubmit){f.requestSubmit();}else{f.submit();}}
 else{el.click();}
}
return {ok:true,text:String(value).slice(0,120)};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsScroll 窗口滚动或容器内滚动。
const jsScroll = `(function(){try{
var op='gaeaScroll'; var dir=%s; var amt=%d; var container=%s;
var delta=dir==='up'?-amt:amt;
if(container){
 var c=document.querySelector(container);
 if(!c)return {ok:false,error:"未找到元素："+container};
 c.scrollBy(0,delta);
 return {ok:true,top:String(c.scrollTop)};
}
window.scrollBy(0,delta);
return {ok:true,top:String(window.scrollY)};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`
