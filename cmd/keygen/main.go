package main

import (
	"fmt"
	"os"
	"time"

	"wechat-obsidian-bot/internal/license"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("License Key 生成器")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Println("  keygen <机器码> <过期日期>")
		fmt.Println("  keygen <机器码> permanent")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  keygen A1B2C3D4E5F6 2026-12-31")
		fmt.Println("  keygen A1B2C3D4E5F6 permanent")
		os.Exit(1)
	}

	machineID := os.Args[1]
	expiry := os.Args[2]

	if expiry != "permanent" {
		if _, err := time.Parse("2006-01-02", expiry); err != nil {
			fmt.Printf("日期格式错误: %s (应为 YYYY-MM-DD)\n", expiry)
			os.Exit(1)
		}
	}

	key := license.GenerateKey(machineID, expiry)

	fmt.Println()
	fmt.Printf("机器码: %s\n", machineID)
	fmt.Printf("过期日: %s\n", expiry)
	fmt.Println()
	fmt.Println(key)
	fmt.Println()
}
