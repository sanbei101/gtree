package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	maxFiles int
	maxDepth int

	rootCmd = &cobra.Command{
		Use:   "btree [目录]",
		Short: "以树形结构展示目录内容",
		Long: `btree 是一个树形目录展示工具,支持限制深度和文件数量。

示例:
  btree               # 展示当前目录
  btree /path/to/dir  # 展示指定目录
  btree -n 10 -depth 3`,
		Args: cobra.MaximumNArgs(1),
		Run:  runRoot,
	}

	completionCmd = &cobra.Command{
		Use:   "completion [bash|fish]",
		Short: "为指定 shell 生成自动补全脚本",
		Long: `为 btree 生成自动补全脚本。

安装方法:

  bash:
    source <(btree completion bash)
    或永久生效：btree completion bash >> ~/.bashrc

  fish:
    btree completion fish > ~/.config/fish/completions/btree.fish
`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "fish"},
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				err = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 生成补全脚本失败: %v\n", err)
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&maxFiles, "n", "n", 3, "每个文件夹最多展示的文件数量")
	rootCmd.PersistentFlags().IntVar(&maxDepth, "depth", 5, "最大遍历深度")

	rootCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	}

	rootCmd.AddCommand(completionCmd)
}

func runRoot(cmd *cobra.Command, args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	if targetDir == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = home
		}
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 路径解析失败: %v\n", err)
		os.Exit(1)
	}

	maxWorkers := runtime.NumCPU() * 4
	sem := make(chan struct{}, maxWorkers)

	rootNode := scanDirConcurrent(absPath, maxFiles, maxDepth, 0, sem)

	writer := bufio.NewWriterSize(os.Stdout, 64*1024)
	defer writer.Flush()

	if rootNode != nil {
		printTree(writer, rootNode, "")
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
