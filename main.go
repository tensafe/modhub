package main

import (
	"embed"
	"fmt"
	"log"
	"modhub/route"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// 使用 embed 将静态文件夹打包
//
//go:embed coreruleset/*
var corerulesetFiles embed.FS

// 提取 embed.FS 中的文件到目标目录
func extractEmbedFiles(targetDir, embedRoot string) error {
	// 删除目标目录（如果存在）
	if err := deleteFolder(targetDir); err != nil {
		log.Printf("Error deleting folder %s: %v", targetDir, err)
		return err
	}
	return fsWalk(corerulesetFiles, embedRoot, func(path string, data []byte) error {
		// 去掉嵌套的前缀目录（比如 "coreruleset"）
		relativePath := strings.TrimPrefix(path, embedRoot+"/")

		// 构造输出文件路径
		outputPath := filepath.Join(targetDir, relativePath)

		// 创建必要的父目录
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(outputPath), err)
		}

		// 写入文件
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		return nil
	})
}

// 遍历 embed.FS 并对每个文件执行操作
func fsWalk(fs embed.FS, root string, fn func(path string, data []byte) error) error {
	entries, err := fs.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			// 如果是目录，递归遍历
			if err := fsWalk(fs, path, fn); err != nil {
				return err
			}
		} else {
			// 如果是文件，读取文件内容并执行操作
			data, err := fs.ReadFile(path)
			if err != nil {
				return err
			}

			if err := fn(path, data); err != nil {
				return err
			}
		}
	}

	return nil
}

// 删除文件夹及其所有内容
func deleteFolder(folder string) error {
	// 检查文件夹是否存在
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		// 文件夹不存在，直接返回
		return nil
	}

	// 删除文件夹及其所有内容
	if err := os.RemoveAll(folder); err != nil {
		return fmt.Errorf("failed to delete folder %s: %w", folder, err)
	}

	log.Printf("Deleted folder: %s", folder)
	return nil
}

func main() {
	targetDir := "tswaf_coreruleset"
	// 将所有嵌入的文件释放到目标目录
	if err := extractEmbedFiles(targetDir, "coreruleset"); err != nil {
		log.Printf("Error extracting embedded files: %v", err)
	}

	route.RouterApi()
	// 捕捉系统信号，以便优雅地关闭程序
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号以便优雅地退出
	<-sigChan
}
