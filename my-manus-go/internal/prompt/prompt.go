package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store 提供提示词模板的加载与占位符填充
type Store struct {
	dir string
}

// Load 校验提示词目录存在
func Load(dir string) (*Store, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("提示词目录不存在: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Get 读取提示词文件原始内容（文件名 = 名称 + ".txt"）
func (s *Store) Get(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(s.dir, name+".txt"))
	if err != nil {
		return "", fmt.Errorf("读取提示词 %s: %w", name, err)
	}
	return string(b), nil
}

// Render 填充 {key} 占位符（对应 prompt 里的 {task}/{agentData}/{subTaskChain} 等）
func Render(tpl string, vars map[string]string) string {
	s := tpl
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
