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
		t.Error("❌ 扫描结果返回了 nil, 不合预期")
	} else if len(rootNode.SubDirs) != wideCount {
		t.Errorf("❌ 期望第一层有 %d 个子目录, 实际得到 %d 个", wideCount, len(rootNode.SubDirs))
	} else {
		t.Log("✅ 压测成功！未发生任何死锁，平稳穿透大并发目录。")
	}
}
func BenchmarkScanDirConcurrent(b *testing.B) {
	benchmarks := []struct {
		name        string // 分组标签
		depth       int    // 嵌套最大深度
		breadth     int    // 每层分裂出的子目录数
		filesPerDir int    // 每个目录下生成的桩文件数
	}{
		{name: "Scale=Small", depth: 1, breadth: 10, filesPerDir: 5},      // 常规小目录
		{name: "Scale=Wide", depth: 1, breadth: 1000, filesPerDir: 2},     // 极宽单层
		{name: "Scale=Deep", depth: 20, breadth: 1, filesPerDir: 3},       // 极深单线嵌套
		{name: "Scale=Large", depth: 2, breadth: 25, filesPerDir: 100},    // 大规模混合树
	}

	maxWorkers := runtime.NumCPU() * 4

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			tmpDir, err := os.MkdirTemp("", "gtree_bench_*")
			if err != nil {
				b.Fatalf("无法创建基准测试临时目录: %v", err)
			}
			defer os.RemoveAll(tmpDir)
			var buildTree func(string, int)
			buildTree = func(currentPath string, currentDepth int) {
				if currentDepth > bm.depth {
					return
				}
				for k := range bm.filesPerDir {
					filePath := filepath.Join(currentPath, fmt.Sprintf("mock_file_%d.log", k))
					os.WriteFile(filePath, []byte("perf_data"), 0644)
				}
				if currentDepth < bm.depth {
					for j := range bm.breadth {
						subDir := filepath.Join(currentPath, fmt.Sprintf("sub_dir_%d", j))
						if err := os.Mkdir(subDir, 0755); err == nil {
							buildTree(subDir, currentDepth+1)
						}
					}
				}
			}
			buildTree(tmpDir, 0)

			b.ResetTimer()

			for b.Loop() {
				sem := make(chan struct{}, maxWorkers)
				node := scanDirConcurrent(tmpDir, 2, 30, 0, sem)
				if node == nil {
					b.Fatal("基准测试运行失败,返回了空的节点")
				}
			}
		})
	}
}
