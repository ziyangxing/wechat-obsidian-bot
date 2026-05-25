package media

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadFromURL downloads media from a URL to the specified directory.
func DownloadFromURL(url, destDir, filename string) (string, error) {
	if filename == "" {
		filename = fmt.Sprintf("media_%d", time.Now().UnixNano())
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	ext := detectExt(resp.Header.Get("Content-Type"), url)
	filePath := filepath.Join(destDir, filename+ext)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return filePath, nil
}

func detectExt(contentType string, url string) string {
	switch {
	case strings.Contains(contentType, "image/jpeg") || strings.Contains(contentType, "image/jpg"):
		return ".jpg"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/webp"):
		return ".webp"
	case strings.Contains(contentType, "video/mp4"):
		return ".mp4"
	case strings.Contains(contentType, "audio/mpeg") || strings.Contains(contentType, "audio/mp3"):
		return ".mp3"
	case strings.Contains(contentType, "audio/amr"):
		return ".amr"
	default:
		// Try to detect from URL (strip query params first)
		cleanURL := strings.SplitN(url, "?", 2)[0]
		if ext := filepath.Ext(cleanURL); ext != "" && len(ext) <= 5 {
			return ext
		}
		// Check wx_fmt query param for WeChat images
		if idx := strings.Index(url, "wx_fmt="); idx >= 0 {
			fmt := url[idx+7:]
			if semiIdx := strings.IndexAny(fmt, "&; "); semiIdx > 0 {
				fmt = fmt[:semiIdx]
			}
			switch fmt {
			case "jpeg", "jpg":
				return ".jpg"
			case "png":
				return ".png"
			case "gif":
				return ".gif"
			case "webp":
				return ".webp"
			}
		}
		return ".bin"
	}
}
