package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScanDirConcurrent_DeadlockPrevention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtree_test_*")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	wideCount := 100
	deepCount := 4

	for i := range wideCount {
		currentPath := filepath.Join(tmpDir, fmt.Sprintf("wide_dir_%d", i))
		for j := range deepCount {
			currentPath = filepath.Join(currentPath, fmt.Sprintf("deep_dir_%d", j))
		}
		if err := os.MkdirAll(currentPath, 0755); err != nil {
			t.Fatalf("生成桩目录失败: %v", err)
		}
		err = os.WriteFile(filepath.Join(currentPath, "evidence.txt"), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("生成桩文件失败: %v", err)
		}
	}

	sem := make(chan struct{}, 2)

	t.Log("🚀 开始压测高密度树状结构...")
	rootNode := scanDirConcurrent(tmpDir, 2, 5, 0, sem)

	if rootNode == nil {
		t.Error("❌ 扫描结果返回了 nil，不合预期")
	} else if len(rootNode.SubDirs) != wideCount {
		t.Errorf("❌ 期望第一层有 %d 个子目录, 实际得到 %d 个", wideCount, len(rootNode.SubDirs))
	} else {
		t.Log("✅ 压测成功！未发生任何死锁，平稳穿透大并发目录。")
	}
}
func BenchmarkScanDirConcurrent(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "gtree_bench_*")
	if err != nil {
		b.Fatalf("无法创建基准测试临时目录: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := range 50 {
		p1 := filepath.Join(tmpDir, fmt.Sprintf("parent_dir_%d", i))
		for j := range 10 {
			p2 := filepath.Join(p1, fmt.Sprintf("child_dir_%d", j))
			if err := os.MkdirAll(p2, 0755); err != nil {
				b.Fatal(err)
			}
			for k := range 10 {
				filePath := filepath.Join(p2, fmt.Sprintf("mock_file_%d.log", k))
				if err := os.WriteFile(filePath, []byte("performance_test"), 0644); err != nil {
					b.Fatal(err)
				}
			}
		}
	}

	maxWorkers := runtime.NumCPU() * 4

	for b.Loop() {
		sem := make(chan struct{}, maxWorkers)
		node := scanDirConcurrent(tmpDir, 2, 4, 0, sem)
		if node == nil {
			b.Fatal("基准测试运行失败，返回了空的节点")
		}
	}
}
