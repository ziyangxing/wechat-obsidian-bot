package article

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ArticleResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
	Author  string `json:"author"`
	Error   string `json:"error"`
}

type VoiceResult struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

var extractScript string
var transcribeScript string

func init() {
	// Try multiple locations for the Python scripts
	searchPaths := []string{}

	// 1. Executable directory (for release builds)
	if exe, err := os.Executable(); err == nil {
		searchPaths = append(searchPaths, filepath.Dir(exe))
	}

	// 2. Current working directory
	if wd, err := os.Getwd(); err == nil {
		searchPaths = append(searchPaths, wd)
		// 3. Parent of cwd (in case binary is in a subdirectory)
		searchPaths = append(searchPaths, filepath.Dir(wd))
	}

	// 4. Project root (hardcoded fallback)
	searchPaths = append(searchPaths, `D:\item\Claude_code_build\knowledge_m\wechat-obsidian-bot`)

	for _, dir := range searchPaths {
		candidate := filepath.Join(dir, "extract_article.py")
		if fileExists(candidate) {
			extractScript = candidate
			break
		}
	}
	if extractScript == "" {
		extractScript = "extract_article.py"
	}

	for _, dir := range searchPaths {
		candidate := filepath.Join(dir, "transcribe_voice.py")
		if fileExists(candidate) {
			transcribeScript = candidate
			break
		}
	}
	if transcribeScript == "" {
		transcribeScript = "transcribe_voice.py"
	}

	fmt.Fprintf(os.Stderr, "[article] extract_script=%s\n", extractScript)
	fmt.Fprintf(os.Stderr, "[article] transcribe_script=%s\n", transcribeScript)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Extract(url string) (*ArticleResult, error) {
	output, err := runScript(extractScript, url)
	if err != nil {
		return nil, err
	}

	var result ArticleResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("解析提取结果失败: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("提取文章失败: %s", result.Error)
	}
	if result.Title == "" && result.Content == "" {
		return nil, fmt.Errorf("文章提取返回空内容")
	}
	return &result, nil
}

func TranscribeVoice(filePath string) (*VoiceResult, error) {
	output, err := runScript(transcribeScript, filePath)
	if err != nil {
		return nil, err
	}

	var result VoiceResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("解析转录结果失败: %w", err)
	}
	return &result, nil
}

func runScript(script string, args ...string) (string, error) {
	cmd := exec.Command("python", append([]string{script}, args...)...)

	// Set working directory to script location
	cmd.Dir = filepath.Dir(script)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		return "", fmt.Errorf("脚本执行失败 (exit=%v): stderr=%s stdout=%s",
			err, truncateStr(stderrStr, 500), truncateStr(stdoutStr, 200))
	}

	// Strip loguru noise - find the JSON line starting with {
	outputStr := stdoutStr
	if idx := strings.LastIndex(outputStr, `{"title"`); idx >= 0 {
		outputStr = outputStr[idx:]
	} else if idx := strings.LastIndex(outputStr, `{"error"`); idx >= 0 {
		outputStr = outputStr[idx:]
	} else if idx := strings.LastIndex(outputStr, "{"); idx >= 0 {
		outputStr = outputStr[idx:]
	}

	return outputStr, nil
}

func truncateStr(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func IsArticleURL(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.Contains(trimmed, "mp.weixin.qq.com/s/") ||
		strings.Contains(trimmed, "mp.weixin.qq.com/s?")
}
