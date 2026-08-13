package app

// Herdsman 模型库磁盘治理（E1-4）：已装模型占用汇总 + 数据目录所在卷的容量/余量，
// 供模型库 KPI 展示，避免 110GB 级模型堆叠导致磁盘爆满无感知。

import (
	"errors"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// diskFreeFn 返回 path 所在卷的 total/free 字节（包级变量，测试可注入替身）。
var diskFreeFn = func(path string) (total, free uint64, err error) {
	ptr, perr := windows.UTF16PtrFromString(path)
	if perr != nil {
		return 0, 0, perr
	}
	var freeBytesAvail, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(ptr, &freeBytesAvail, &totalBytes, &totalFree); err != nil {
		return 0, 0, err
	}
	return totalBytes, freeBytesAvail, nil
}

// herdsmanDiskInfo 计算 Herdsman 数据目录所在卷的总/余空间（字节）。
// 目录无法定位或探测失败时返回错误（调用方降级为 0 + 提示文案）。
func herdsmanDiskInfo() (total, free int64, err error) {
	dir := herdsmanDataDir()
	if dir == "" {
		return 0, 0, errors.New("无法定位 Herdsman 数据目录")
	}
	vol := filepath.VolumeName(dir)
	if vol == "" {
		vol = dir
	} else {
		vol += `\`
	}
	t, f, derr := diskFreeFn(vol)
	if derr != nil {
		return 0, 0, derr
	}
	const maxInt64 = uint64(1<<63 - 1)
	if t > maxInt64 || f > maxInt64 {
		return 0, 0, errors.New("磁盘容量超出 int64 表示范围")
	}
	return int64(t), int64(f), nil
}
