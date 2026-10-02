package dag

// JSON 档文件存储内核（审计 GA2-06 收敛）：Store 与 TemplateStore 原各持一份
// 近逐字副本（safeID→校验→MkdirAll→MarshalIndent→写 .tmp→RenameWithRetry，
// 连空目录语义注释都抄两遍），本文件合一为 fileStore[T]，两 Store 只留各自
// 校验钩子与错误文案名词。行为冻结：方法级语义（含错误文案逐字、空目录
// 非 nil 空集）与收敛前一致。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// fileStore 是档记录 T（Run/Template）的目录存储内核（无状态；并发读改写由
// 调用方串行化）。
//
// 空目录语义（v4.234 真机走查实锤，两处原文共享）：目录不存在=空集——返回
// []T{} 非 nil。nil slice 经 JSON 序列化成 null，会把前端非空断言的消费方
// （DagPanel tpls.length）炸掉；目录存在但全是损坏/非 .json 文件时仍返回
// nil slice（历史口径，List 追加式构建，本刀不改）。
type fileStore[T any] struct {
	dir        string         // 档目录（不存在时 Save 自动建）
	label      string         // 错误文案名词（「流水线」/「模板」）
	id         func(T) string // 记录 id（文件名 stem，safeID 校验对象）
	createdAt  func(T) string // List 排序键（RFC3339 字符串直接比字典序=时间倒序）
	beforeSave func(*T) error // 落盘前钩子（可改写字段）；nil=无
}

func (s fileStore[T]) path(id string) string { return filepath.Join(s.dir, id+".json") }

// save 原子落盘（临时文件+改名）：safeID → beforeSave 钩子 → 建目录 → 序列化。
// 钩子先于 MkdirAll（TemplateStore 校验语义：坏形状不产生半途目录）。
func (s fileStore[T]) save(v T) error {
	if err := safeID(s.id(v)); err != nil {
		return err
	}
	if s.beforeSave != nil {
		if err := s.beforeSave(&v); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("创建%s目录: %w", s.label, err)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path(s.id(v)) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return fileutil.RenameWithRetry(tmp, s.path(s.id(v)))
}

// get 读单条；不存在报错（fail-closed，不静默空值）。
func (s fileStore[T]) get(id string) (T, error) {
	if err := safeID(id); err != nil {
		var zero T
		return zero, err
	}
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		var zero T
		return zero, fmt.Errorf("读取%s %s: %w", s.label, id, err)
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		var zero T
		return zero, fmt.Errorf("解析%s %s: %w", s.label, id, err)
	}
	return v, nil
}

// list 全部档（创建时间倒序）；损坏文件跳过（只读兼容）。
func (s fileStore[T]) list() ([]T, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		// 目录不存在=空集语义：nil slice 经 JSON 序列化成 null，会把前端
		// 非空断言的消费方（DagPanel tpls.length）炸掉（v4.234 真机走查实锤）。
		return []T{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []T
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var v T
		if json.Unmarshal(b, &v) == nil && s.id(v) != "" {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return s.createdAt(out[i]) > s.createdAt(out[j]) })
	return out, nil
}

// del 删档即弃；不存在报错（fail-closed，不静默成功）。
func (s fileStore[T]) del(id string) error {
	if err := safeID(id); err != nil {
		return err
	}
	if err := os.Remove(s.path(id)); err != nil {
		return fmt.Errorf("删除%s %s: %w", s.label, id, err)
	}
	return nil
}
