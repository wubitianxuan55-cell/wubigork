//go:build windows

package computer

// UIA（UI Automation COM）树遍历 → treeNode（施工单 W2）。
// 纪律：不引第三方库，ole32 CoCreateInstance + 手写 vtable 调用。
// vtable 序与属性 ID 一律取自本机 Windows SDK 10.0.26100 UIAutomationClient.h
// 的 C++ 类声明序（=vtable 序），不凭记忆：
//   IUIAutomation: ElementFromHandle=6 get_ControlViewWalker=13
//   IUIAutomationTreeWalker: GetFirstChildElement=4 GetNextSiblingElement=6
//   IUIAutomationElement: GetCurrentPropertyValue=10 LocalizedControlType=22
//                         Name=23 AutomationId=29 HelpText=31
//   属性 ID：ExpandCollapse 可用=30028 Invoke 可用=30031 Scroll 可用=30034
//            SelectionItem 可用=30036 Value 可用=30043 ValueValue=30045
//            ValueIsReadOnly=30046 ExpandCollapseState=30070
//            SelectionItemIsSelected=30079
// COM 对象指针全程 unsafe.Pointer（GC 不可见外部内存；vet unsafeptr 干净），
// 只在系统调用实参处转 uintptr。

import (
	"crypto/rand"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modOle32    = syscall.NewLazyDLL("ole32.dll")
	modOleAut32 = syscall.NewLazyDLL("oleaut32.dll")

	procCoInitializeEx   = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize   = modOle32.NewProc("CoUninitialize")
	procCoCreateInstance = modOle32.NewProc("CoCreateInstance")
	procSysFreeString    = modOleAut32.NewProc("SysFreeString")
	procLstrlenW         = modKernelW.NewProc("lstrlenW")
)

const (
	clsidCUIAutomation = "{ff48dba4-60ef-4201-aa87-54103eef594e}"
	iidIUIAutomation   = "{30cbe57d-d9d0-452a-ab13-7ac5ac4825ee}"

	coinitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1
	sFalseCode              = 0x1        // CoInitializeEx S_FALSE=已初始化（仍须配对 Uninit）
	rpcEChangedMode         = 0x80010106 // 线程已有别种套间：不拥有初始化，照样可用

	vtblRelease = 2

	// IUIAutomation
	autoElementFromHandle = 6
	autoControlViewWalker = 14 // CreateTreeWalker=13 之后；错序调用触发 CFG fastfail=进程无痕死亡

	// IUIAutomationTreeWalker
	walkerFirstChild  = 4
	walkerNextSibling = 6

	// IUIAutomationElement
	elemGetCurrentPropertyValue = 10
	elemLocalizedControlType    = 22
	elemName                    = 23
	elemAutomationId            = 29
	elemHelpText                = 31

	propExpandCollapseAvail = 30028
	propInvokeAvail         = 30031
	propScrollAvail         = 30034
	propSelectionItemAvail  = 30036
	propValueAvail          = 30043
	propValueValue          = 30045
	propValueIsReadOnly     = 30046
	propExpandCollapseState = 30070
	propSelectionItemIsSel  = 30079

	maxTreeDepth = 60
	maxTreeNodes = 8000
)

// oleVariant 与 Windows VARIANT 对齐（x64=24 字节）。union 用 [8]byte 原样
// 存放，读取时按 vt 解码（指针的指针读出成员值，vet 干净）。
type oleVariant struct {
	VT    uint16
	Res   [3]uint16
	Union [8]byte
	Extra [8]byte
}

const (
	vtEmpty = 0
	vtI4    = 3
	vtBSTR  = 8
	vtBool  = 11
)

func (v oleVariant) asBool() bool {
	// VARIANT_BOOL 是 16 位：必须按 int16 读，int32 会把高位垃圾字节读成真。
	return v.VT == vtBool && *(*uint16)(unsafe.Pointer(&v.Union[0])) != 0
}
func (v oleVariant) asI4() int32 { return *(*int32)(unsafe.Pointer(&v.Union[0])) }
func (v oleVariant) asBSTR() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&v.Union[0]))
}

// uiaCtx 一次观察的 COM 作用域：auto/walker 句柄 + 初始化归属 + 计数护栏。
type uiaCtx struct {
	auto     unsafe.Pointer
	walker   unsafe.Pointer
	nodes    int
	ownsInit bool
}

func vcall(com unsafe.Pointer, idx int, args ...uintptr) uintptr {
	vtbl := (*[64]uintptr)(*(*unsafe.Pointer)(com)) // COM 对象头=IUnknown*→vtable
	// Windows 上 SyscallN 返回 (r1, r2, err)；COM 方法只消费 r1（HRESULT）。
	r1, _, _ := syscall.SyscallN(vtbl[idx], append([]uintptr{uintptr(com)}, args...)...)
	return r1
}

func release(com unsafe.Pointer) {
	if com != nil {
		vcall(com, vtblRelease)
	}
}

// uiaSession 建立 COM 作用域（观察结束后必须 close）。
func uiaSession() (*uiaCtx, error) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	u := &uiaCtx{ownsInit: hr == 0 || hr == sFalseCode}
	if !u.ownsInit && hr != rpcEChangedMode {
		return nil, fmt.Errorf("computer: CoInitializeEx hr=0x%x", hr)
	}
	clsid, err := guidFromString(clsidCUIAutomation)
	if err == nil {
		iid, err2 := guidFromString(iidIUIAutomation)
		err = err2
		if err == nil {
			hr, _, _ = procCoCreateInstance.Call(
				uintptr(unsafe.Pointer(&clsid)), 0, clsctxInprocServer,
				uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&u.auto)))
			if hr == 0 && u.auto != nil {
				if hr := vcall(u.auto, autoControlViewWalker, uintptr(unsafe.Pointer(&u.walker))); hr == 0 && u.walker != nil {
					return u, nil
				}
				err = fmt.Errorf("computer: get_ControlViewWalker hr=0x%x", hr)
			} else {
				err = fmt.Errorf("computer: CoCreateInstance(CUIAutomation) hr=0x%x（UIA 不可用）", hr)
			}
		}
	}
	u.close()
	return nil, err
}

func (u *uiaCtx) close() {
	release(u.walker)
	release(u.auto)
	if u.ownsInit {
		procCoUninitialize.Call()
	}
}

// elementFromHWND 取窗口的 UIA 元素。
func (u *uiaCtx) elementFromHWND(hwnd uintptr) (unsafe.Pointer, error) {
	var elem unsafe.Pointer
	hr := vcall(u.auto, autoElementFromHandle, hwnd, uintptr(unsafe.Pointer(&elem)))
	if hr != 0 || elem == nil {
		return nil, fmt.Errorf("computer: ElementFromHandle hr=0x%x", hr)
	}
	return elem, nil
}

// bstrProp 读 BSTR 字符串属性（get_Current* 形态，out=BSTR*）。
func bstrProp(elem unsafe.Pointer, vidx int) string {
	var bstr unsafe.Pointer
	hr := vcall(elem, vidx, uintptr(unsafe.Pointer(&bstr)))
	if hr != 0 || bstr == nil {
		return ""
	}
	defer procSysFreeString.Call(uintptr(bstr))
	return bstrToString(bstr)
}

// bstrToString 拷贝 BSTR（UTF-16 → Go string）。长度经 lstrlenW 读
// （BSTR 以 NUL 结尾；避开头部指针算术，vet-clean）。
func bstrToString(bstr unsafe.Pointer) string {
	if bstr == nil {
		return ""
	}
	n, _, _ := procLstrlenW.Call(uintptr(bstr))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(bstr), n))
}

// propVariant 读属性 VARIANT（失败返回空 VARIANT，vtEmpty）。
func propVariant(elem unsafe.Pointer, propID uint32) oleVariant {
	var v oleVariant
	hr := vcall(elem, elemGetCurrentPropertyValue, uintptr(propID), uintptr(unsafe.Pointer(&v)))
	if hr != 0 {
		return oleVariant{VT: vtEmpty}
	}
	return v
}

// elementToNode 单节点 → treeNode：角色/名字/ID/描述用 typed getter；
// 状态与 Secondary Actions 用「先试 pattern 可用、再取状态属性」两步；
// 值模式：非只读 → (settable, string)，值进 Value 后缀（对齐 codex 样本
// 「62 编辑 (settable, string) 搜索会话名称」）。
func (u *uiaCtx) elementToNode(elem unsafe.Pointer) *treeNode {
	n := &treeNode{
		Role:        bstrProp(elem, elemLocalizedControlType),
		Name:        bstrProp(elem, elemName),
		ID:          bstrProp(elem, elemAutomationId),
		Description: bstrProp(elem, elemHelpText),
	}
	if variantOK(propVariant(elem, propExpandCollapseAvail)) {
		switch propVariant(elem, propExpandCollapseState).asI4() {
		case 0:
			n.States = append(n.States, "expanded")
			n.Secondary = append(n.Secondary, "Collapse")
		case 1:
			n.States = append(n.States, "collapsed")
			n.Secondary = append(n.Secondary, "Expand")
		}
	}
	if variantOK(propVariant(elem, propSelectionItemAvail)) {
		n.States = append(n.States, "selectable")
		if propVariant(elem, propSelectionItemIsSel).asBool() {
			n.States = append(n.States, "selected")
		}
	}
	if vv := propVariant(elem, propValueValue); vv.VT == vtBSTR {
		bstr := vv.asBSTR()
		n.Value = bstrToString(bstr)
		procSysFreeString.Call(uintptr(bstr)) // 读毕即还（含空串路径，零泄漏）
		if !variantOK(propVariant(elem, propValueIsReadOnly)) {
			n.States = append(n.States, "settable", "string")
		}
	}
	if variantOK(propVariant(elem, propInvokeAvail)) {
		n.Secondary = append(n.Secondary, "Raise")
	}
	if variantOK(propVariant(elem, propScrollAvail)) {
		n.Secondary = append(n.Secondary, "Scroll Up", "Scroll Down")
	}
	return n
}

// variantOK VARIANT 是否为可用布尔（pattern-available 探测）。
func variantOK(v oleVariant) bool { return v.VT == vtBool && v.asBool() }

// walk 深度先序遍历（ControlViewWalker）；护栏：深度/节点数双帽。
func (u *uiaCtx) walk(elem unsafe.Pointer, depth int) *treeNode {
	if depth > maxTreeDepth || u.nodes >= maxTreeNodes {
		return nil
	}
	n := u.elementToNode(elem)
	u.nodes++
	var child unsafe.Pointer
	hr := vcall(u.walker, walkerFirstChild, uintptr(unsafe.Pointer(elem)), uintptr(unsafe.Pointer(&child)))
	for hr == 0 && child != nil && u.nodes < maxTreeNodes {
		if c := u.walk(child, depth+1); c != nil {
			n.Children = append(n.Children, c)
		}
		var next unsafe.Pointer
		hrNext := vcall(u.walker, walkerNextSibling, uintptr(unsafe.Pointer(child)), uintptr(unsafe.Pointer(&next)))
		release(child)
		child, hr = next, hrNext
	}
	release(child)
	return n
}

// uiaTreeText 观察入口：HWND → UIA ControlView 树 → codex 文法文本。
func uiaTreeText(hwnd uintptr, title, app string) (string, error) {
	u, err := uiaSession()
	if err != nil {
		return "", err
	}
	defer u.close()
	elem, err := u.elementFromHWND(hwnd)
	if err != nil {
		return "", err
	}
	defer release(elem)
	root := u.walk(elem, 0)
	win := Window{ID: int(int32(hwnd)), Title: title, App: app}
	return RenderTree(win, root), nil
}

// newUUID v4 形态 shot id（每次观察更换，对齐 codex 截图 id 语义）。
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
