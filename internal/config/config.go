package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	VaultPath        string `json:"vault_path"`
	InboxFolder      string `json:"inbox_folder"`
	DownloadMedia    bool   `json:"download_media"`
	ExtractArticles  bool   `json:"extract_articles"`
	ArticleExtractor string `json:"article_extractor"`
	AutoFriendAdd    bool   `json:"auto_friend_add"`
	FriendWelcomeMsg string `json:"friend_welcome_msg"`
	LicenseKey       string `json:"license_key"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		InboxFolder:      "微信收件箱",
		DownloadMedia:    true,
		ExtractArticles:  true,
		ArticleExtractor: "feedgrab",
		AutoFriendAdd:    true,
		FriendWelcomeMsg: "你好，我是 Obsidian 助手。把任何内容转发给我，我会自动同步到你的 Obsidian 知识库。",
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
