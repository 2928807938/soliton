package generator

import (
	"path/filepath"
	"strings"
)

// toLowerFirst 将字符串首字母转为小写
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

// calculateImportPath 根据模块信息计算目录的完整 import 路径
// moduleName: Go 模块名，如 "mymodule"
// moduleRoot: 模块根目录绝对路径
// targetDir: 目标目录绝对路径
func calculateImportPath(moduleName, moduleRoot, targetDir string) string {
	if moduleName == "" || moduleRoot == "" {
		return filepath.Base(targetDir)
	}

	// 计算相对路径
	relPath, err := filepath.Rel(moduleRoot, targetDir)
	if err != nil {
		return filepath.Base(targetDir)
	}

	// 将路径分隔符统一为 /
	relPath = filepath.ToSlash(relPath)

	// 如果相对路径是 "."，则直接返回模块名
	if relPath == "." {
		return moduleName
	}

	// 拼接完整的 import 路径
	return moduleName + "/" + relPath
}
