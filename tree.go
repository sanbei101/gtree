package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Node struct {
	Name    string
	Files   []string
	SubDirs []*Node
	HasMore bool
}

func scanDirConcurrent(path string, maxFiles, maxDepth, currentDepth int, sem chan struct{}) *Node {
	if currentDepth > maxDepth {
		return nil
	}

	sem <- struct{}{}

	f, err := os.Open(path)
	if err != nil {
		<-sem
		return nil
	}

	entries, err := f.ReadDir(-1)
	f.Close()
	<-sem

	if err != nil {
		return nil
	}

	node := &Node{
		Name: filepath.Base(path),
	}
	if node.Name == "." || node.Name == "/" || node.Name == "" {
		node.Name = path
	}

	var dirEntries []os.DirEntry
	var fileNames []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirEntries = append(dirEntries, entry)
		} else {
			fileNames = append(fileNames, entry.Name())
		}
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		return dirEntries[i].Name() < dirEntries[j].Name()
	})
	sort.Strings(fileNames)

	if len(fileNames) > maxFiles {
		node.Files = fileNames[:maxFiles]
		node.HasMore = true
	} else {
		node.Files = fileNames
	}

	if len(dirEntries) > 0 && currentDepth < maxDepth {
		node.SubDirs = make([]*Node, len(dirEntries))
		var wg sync.WaitGroup

		for i, dirEntry := range dirEntries {
			subPath := filepath.Join(path, dirEntry.Name())
			wg.Add(1)

			go func(idx int, p string) {
				defer wg.Done()
				node.SubDirs[idx] = scanDirConcurrent(p, maxFiles, maxDepth, currentDepth+1, sem)
			}(i, subPath)
		}
		wg.Wait()

		validDirs := make([]*Node, 0, len(node.SubDirs))
		for _, sub := range node.SubDirs {
			if sub != nil {
				validDirs = append(validDirs, sub)
			}
		}
		node.SubDirs = validDirs
	}

	return node
}

func printTree(w io.Writer, node *Node, prefix string) {
	if node == nil {
		return
	}

	if prefix == "" {
		fmt.Fprintf(w, "📁 %s/\n", node.Name)
	}

	numFiles := len(node.Files)
	numDirs := len(node.SubDirs)
	totalItems := numFiles
	if node.HasMore {
		totalItems++
	}
	totalItems += numDirs

	itemIdx := 0

	for _, file := range node.Files {
		itemIdx++
		if itemIdx == totalItems {
			fmt.Fprintf(w, "%s└── %s\n", prefix, file)
		} else {
			fmt.Fprintf(w, "%s├── %s\n", prefix, file)
		}
	}

	if node.HasMore {
		itemIdx++
		if itemIdx == totalItems {
			fmt.Fprintf(w, "%s└── ... (剩余文件已略过)\n", prefix)
		} else {
			fmt.Fprintf(w, "%s├── ... (剩余文件已略过)\n", prefix)
		}
	}

	for _, subDir := range node.SubDirs {
		itemIdx++
		isLast := itemIdx == totalItems

		var nextPrefix string
		if isLast {
			fmt.Fprintf(w, "%s└── %s/\n", prefix, subDir.Name)
			nextPrefix = prefix + "    "
		} else {
			fmt.Fprintf(w, "%s├── %s/\n", prefix, subDir.Name)
			nextPrefix = prefix + "│   "
		}

		printTree(w, subDir, nextPrefix)
	}
}
