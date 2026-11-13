import ctypes, ctypes.wintypes as W, sys
u=ctypes.windll.user32; k=ctypes.windll.kernel32
PROCESS_QUERY_LIMITED_INFORMATION=0x1000
def pname(pid):
    h=k.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION,False,pid)
    if not h: return "?"
    b=ctypes.create_unicode_buffer(1024); n=W.DWORD(1024)
    ok=k.QueryFullProcessImageNameW(h,0,b,ctypes.byref(n)); k.CloseHandle(h)
    return b.value.split("\\")[-1] if ok else "?"
rows=[]
@ctypes.WINFUNCTYPE(ctypes.c_bool, W.HWND, W.LPARAM)
def cb(h,l):
    pid=W.DWORD(); u.GetWindowThreadProcessId(h, ctypes.byref(pid))
    pn=pname(pid.value)
    if "wps" not in pn.lower() and "et" != pn.lower()[:2] and "wpp" != pn.lower()[:3]: return True
    cls=ctypes.create_unicode_buffer(256); u.GetClassNameW(h,cls,256)
    ln=u.GetWindowTextLengthW(h); t=ctypes.create_unicode_buffer(ln+1); u.GetWindowTextW(h,t,ln+1)
    st=u.GetWindowLongW(h,-16); ex=u.GetWindowLongW(h,-20)
    rows.append((int(h),pn,cls.value,t.value,hex(st&0xffffffff),hex(ex&0xffffffff),bool(u.IsWindowVisible(h))))
    return True
u.EnumWindows(cb,0)
print("hwnd\tproc\tclass\ttitle\tstyle\texstyle\tvisible")
for r in rows: print("\t".join(str(x) for x in r))
print("total",len(rows))
