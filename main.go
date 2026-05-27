package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/eatmoreapple/openwechat"

	"wechat-obsidian-bot/internal/config"
	"wechat-obsidian-bot/internal/handler"
	"wechat-obsidian-bot/internal/license"
	"wechat-obsidian-bot/internal/server"
	"wechat-obsidian-bot/internal/setup"
	"wechat-obsidian-bot/internal/writer"
)

func main() {
	// --device-code mode: show machine code and copy to clipboard
	if len(os.Args) > 1 && os.Args[1] == "--device-code" {
		code := license.GetMachineID()
		activateURL := fmt.Sprintf("https://ziyangxing.github.io/wechat-obsidian-bot/activate?code=%s", code)
		fmt.Printf("设备码: %s\n", code)
		fmt.Printf("激活链接: %s\n", activateURL)
		copyToClipboard(code)
		fmt.Println("(设备码已复制到剪贴板)")
		// Auto-open browser
		exec.Command("rundll32", "url.dll,FileProtocolHandler", activateURL).Start()
		fmt.Println("浏览器已打开激活页面...")
		fmt.Println("按任意键关闭...")
		var input string
		fmt.Scanln(&input)
		return
	}

	// --setup mode: interactive install wizard
	if len(os.Args) > 1 && os.Args[1] == "--setup" {
		setup.Run()
		return
	}

	// --serve mode: activation web server
	if len(os.Args) > 1 && os.Args[1] == "--serve" {
		port := "8080"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		fmt.Println("启动激活服务器...")
		if err := server.Start(port); err != nil {
			fmt.Fprintf(os.Stderr, "服务器启动失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "请确保 config.json 存在且格式正确\n")
		os.Exit(1)
	}

	// License check
	if cfg.LicenseKey == "" {
		fmt.Fprintln(os.Stderr, "未配置 License Key，请在 config.json 中填入 license_key")
		license.PrintMachineID()
		os.Exit(1)
	}
	if valid, msg := license.Validate(cfg.LicenseKey); !valid {
		fmt.Fprintf(os.Stderr, "License 验证失败: %s\n", msg)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Println("  WeChat → Obsidian 同步机器人")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("Vault 路径: %s\n", cfg.VaultPath)
	fmt.Printf("收件箱目录: %s\n", cfg.InboxFolder)
	fmt.Printf("自动提取文章: %v\n", cfg.ExtractArticles)
	fmt.Printf("自动通过好友: %v\n", cfg.AutoFriendAdd)
	fmt.Println()

	// Initialize Obsidian writer
	w := writer.New(cfg.VaultPath, cfg.InboxFolder)

	// Initialize message handler
	h := handler.New(w, handler.Config{
		DownloadMedia:   cfg.DownloadMedia,
		ExtractArticles: cfg.ExtractArticles,
		AutoFriendAdd:   cfg.AutoFriendAdd,
		WelcomeMsg:      cfg.FriendWelcomeMsg,
	})

	// Create bot
	bot := openwechat.DefaultBot(openwechat.Desktop)

	// Set message handler
	bot.MessageHandler = h.Handle

	// Set QR code output
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// Set login callback
	bot.LoginCallBack = func(body openwechat.CheckLoginResponse) {
		code, err := body.Code()
		if err == nil && code == openwechat.LoginCodeSuccess {
			fmt.Println("✅ 登录成功！")
			fmt.Println("现在可以向 Obsidian 好友转发任何内容了")
			fmt.Println()
			fmt.Println("支持的内容类型：")
			fmt.Println("  📝 文字 → 闪念笔记")
			fmt.Println("  📷 图片 → 自动保存 + 嵌入")
			fmt.Println("  🎬 视频 → 自动保存")
			fmt.Println("  🎤 语音 → 自动保存")
			fmt.Println("  📄 公众号文章 → 自动提取正文")
			fmt.Println("  💬 聊天记录转发 → 格式化保存")
			fmt.Println("  🔗 链接 → 自动保存")
			fmt.Println()
			fmt.Println("按 Ctrl+C 退出")
		}
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\n正在退出...")
		bot.Logout()
		os.Exit(0)
	}()

	// Login (will show QR code in terminal)
	if err := bot.Login(); err != nil {
		log.Fatalf("登录失败: %v", err)
	}

	// Block forever
	bot.Block()
}

func copyToClipboard(text string) {
	cmd := exec.Command("powershell", "-Command", "Set-Clipboard -Value "+text)
	cmd.Run()
}
