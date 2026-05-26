package setup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wechat-obsidian-bot/internal/license"
)

type SetupConfig struct {
	VaultPath string
	InboxDir  string
	AutoStart bool
}

// Run executes the interactive setup wizard.
func Run() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  WeChat → Obsidian Bot 安装向导")
	fmt.Println("========================================")
	fmt.Println()

	// Step 1: Environment check
	fmt.Println("[1/4] 检查运行环境...")
	fmt.Println()
	pyOK := checkCommand("python", "--version")
	pipOK := checkCommand("pip", "--version")
	ffmpegOK := checkCommand("ffmpeg", "-version")
	pyPkgsOK := checkPythonPackages()
	chromiumOK := checkPlaywrightChromium()

	allOK := true
	printStatus("Python 3.12+", pyOK)
	printStatus("pip", pipOK)
	printStatus("ffmpeg", ffmpegOK)
	printStatus("Python feedgrab + whisper", pyPkgsOK)
	printStatus("Playwright Chromium", chromiumOK)
	if !pyOK || !pipOK || !ffmpegOK || !pyPkgsOK || !chromiumOK {
		allOK = false
	}

	fmt.Println()
	if !allOK {
		fmt.Println("⚠️  部分环境缺失。可以运行以下命令安装:")
		fmt.Println()
		if !pyOK {
			fmt.Println("  安装 Python: https://www.python.org/downloads/")
		}
		if !ffmpegOK {
			fmt.Println("  安装 ffmpeg: pip install ffmpeg-python 或从 https://ffmpeg.org 下载")
		}
		if !pyPkgsOK {
			fmt.Println("  pip install feedgrab openai-whisper")
		}
		if !chromiumOK {
			fmt.Println("  python -m playwright install chromium")
		}
		fmt.Println()
		fmt.Print("环境修复后，重新运行 setup 即可。按 Enter 继续配置...")
		reader.ReadString('\n')
	} else {
		fmt.Println("✅ 环境检查全部通过！")
	}

	fmt.Println()

	// Step 2: Find Obsidian vaults
	fmt.Println("[2/4] 搜索 Obsidian Vault...")
	fmt.Println()
	vaultPath := ""
	vaults := findObsidianVaults()
	if len(vaults) > 0 {
		fmt.Println("找到以下 Vault:")
		for i, v := range vaults {
			fmt.Printf("  %d. %s\n", i+1, v)
		}
		fmt.Println("  0. 手动输入路径")
		fmt.Println()
		fmt.Print("选择 Vault 编号 [1]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" || input == "1" {
			vaultPath = vaults[0]
		} else if choice := atoi(input); choice > 0 && choice <= len(vaults) {
			vaultPath = vaults[choice-1]
		} else if input == "0" {
			vaultPath = promptPath(reader, "")
		} else {
			vaultPath = input
			if !dirExists(vaultPath) {
				vaultPath = promptPath(reader, vaultPath)
			}
		}
	} else {
		vaultPath = promptPath(reader, "")
	}

	fmt.Println()

	// Step 3: Inbox folder
	fmt.Println("[3/4] 同步文件夹设置")
	fmt.Println()
	fmt.Print("文件夹名称 [raw/wechat]: ")
	inbox, _ := reader.ReadString('\n')
	inbox = strings.TrimSpace(inbox)
	if inbox == "" {
		inbox = "raw/wechat"
	}

	fmt.Println()

	// Step 4: License
	fmt.Println("[4/4] License 激活")
	fmt.Println()
	machineID := license.GetMachineID()
	fmt.Printf("你的设备码: %s\n", machineID)
	fmt.Println()
	fmt.Println("请将此设备码发送给作者获取 License Key。")
	fmt.Println()
	fmt.Print("输入 License Key（没有可回车跳过）: ")
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)

	if key != "" {
		ok, msg := license.Validate(key)
		if ok {
			fmt.Println("✅ License 验证通过！")
		} else {
			fmt.Printf("⚠️  License 验证失败: %s\n", msg)
			fmt.Print("继续配置但启动时需要有效的 License Key。按 Enter...")
			reader.ReadString('\n')
		}
	} else {
		fmt.Println("已跳过。启动前请在 config.json 中手动填入 license_key。")
	}

	fmt.Println()

	// Write config
	cfg := map[string]interface{}{
		"vault_path":          vaultPath,
		"inbox_folder":        inbox,
		"download_media":      true,
		"extract_articles":    true,
		"article_extractor":   "feedgrab",
		"auto_friend_add":     true,
		"friend_welcome_msg":  "你好，我是 Obsidian 同步助手。转发任何内容给我，我会自动保存到你的知识库。",
		"license_key":         key,
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile("config.json", data, 0644)

	fmt.Println("========================================")
	fmt.Println("  安装完成！")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("配置已保存: config.json\n")
	fmt.Printf("Vault: %s\n", vaultPath)
	fmt.Printf("同步到: %s/日期/\n", inbox)
	fmt.Println()
	if key != "" {
		fmt.Println("✅ License 已激活")
	}
	fmt.Println()
	fmt.Println("下一步: 双击 '启动.bat' 开始使用")
	fmt.Println()
	fmt.Print("按 Enter 退出...")
	reader.ReadString('\n')
}

func checkCommand(cmd string, args ...string) bool {
	_, err := exec.Command(cmd, args...).Output()
	return err == nil
}

func checkPythonPackages() bool {
	for _, pkg := range []string{"feedgrab", "whisper"} {
		cmd := exec.Command("python", "-c", "import "+pkg)
		if err := cmd.Run(); err != nil {
			return false
		}
	}
	return true
}

func checkPlaywrightChromium() bool {
	// Check if chromium is installed for playwright
	cmd := exec.Command("python", "-c", `
import os, sys
try:
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        browser = p.chromium.launch()
        browser.close()
    sys.exit(0)
except:
    sys.exit(1)
`)
	return cmd.Run() == nil
}

func findObsidianVaults() []string {
	var vaults []string
	home, _ := os.UserHomeDir()
	searchPaths := []string{
		filepath.Join(home, "Documents", "Obsidian"),
		filepath.Join(home, "文档", "Obsidian"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "文档"),
		filepath.Join(home, "OneDrive", "Obsidian"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "桌面"),
		"D:\\Obsidian",
		"D:\\item",
	}

	seen := make(map[string]bool)
	for _, base := range searchPaths {
		if !dirExists(base) {
			continue
		}
		depth := 0
		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if !info.IsDir() {
				return nil
			}
			// Calculate depth relative to base
			rel, _ := filepath.Rel(base, path)
			if rel == "." {
				depth = 0
			} else {
				depth = len(strings.Split(rel, string(filepath.Separator)))
			}
			if depth > 3 {
				return filepath.SkipDir
			}
			// Obsidian vault has a .obsidian folder inside
			obsidianDir := filepath.Join(path, ".obsidian")
			if dirExists(obsidianDir) && !seen[path] {
				seen[path] = true
				vaults = append(vaults, path)
			}
			return nil
		})
	}
	return vaults
}

func printStatus(name string, ok bool) {
	if ok {
		fmt.Printf("  ✅ %s\n", name)
	} else {
		fmt.Printf("  ❌ %s — 未安装\n", name)
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func promptPath(reader *bufio.Reader, prevAttempt string) string {
	if prevAttempt != "" {
		fmt.Printf("路径不存在: %s\n", prevAttempt)
	}
	fmt.Println("请输入 Obsidian Vault 的完整路径")
	fmt.Println("（右键 Obsidian 侧边栏 Vault 名 → 复制路径）")
	fmt.Print("> ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if path != "" && dirExists(path) {
		return path
	}
	if path == "" {
		return prevAttempt // can't do anything, return whatever
	}
	return promptPath(reader, path) // retry
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
