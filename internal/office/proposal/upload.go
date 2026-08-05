package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveUploadedFile 保存前端上传的原始文件字节流，返回更新后的方案。
// 文件写入方案数据目录 uploads/<proposalID>/ 下，供 ConvertFiles 调用转换器处理。
func (s *Service) SaveUploadedFile(proposalID, fileName string, data []byte) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if fileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}
	if p.BidSummary == nil {
		p.BidSummary = &BidSummary{Extra: make(map[string]string)}
	}
	dir := filepath.Join(s.store.FilesDir(), proposalID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}
	path := uniquePath(filepath.Join(dir, sanitizeFileName(fileName)))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}
	if err := s.store.AddFile(proposalID, "attachment", filepath.Base(path), path, len(data)); err != nil {
		return nil, err
	}
	p.BidSummary.RawFiles = append(p.BidSummary.RawFiles, FileDoc{
		Name: filepath.Base(path), Path: path, Size: len(data),
	})
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	replacer := strings.NewReplacer("/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "upload.bin"
	}
	return name
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}
