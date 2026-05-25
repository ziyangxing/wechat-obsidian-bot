package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Writer struct {
	VaultPath   string
	InboxFolder string
}

type Note struct {
	Title      string
	Content    string
	Source     string // 来源：微信好友昵称
	MediaFiles []string
	Tags       []string
	NoteType   string // text, image, video, article, chat_record
	URL        string // 原文链接（文章类型）
}

func New(vaultPath, inboxFolder string) *Writer {
	return &Writer{
		VaultPath:   vaultPath,
		InboxFolder: inboxFolder,
	}
}

func (w *Writer) WriteNote(note *Note) (string, error) {
	now := time.Now()
	dateDir := filepath.Join(w.VaultPath, w.InboxFolder, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return "", fmt.Errorf("创建日期目录失败: %w", err)
	}

	filename := w.generateFilename(now, note)
	filePath := filepath.Join(dateDir, filename)

	fm := w.buildFrontmatter(now, note)
	body := note.Content

	fullContent := fm + "\n" + body

	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return filePath, nil
}

func (w *Writer) UpdateNoteContent(filePath, newContent string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	content := string(data)
	// Split at second "---" (end of frontmatter)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return fmt.Errorf("文件格式不正确，找不到 frontmatter 结束标记")
	}

	// Reconstruct: frontmatter + new body
	result := parts[0] + "---" + parts[1] + "---\n" + newContent

	if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

func (w *Writer) AttachmentsDir() string {
	now := time.Now()
	dir := filepath.Join(w.VaultPath, w.InboxFolder, now.Format("2006-01-02"), "attachments")
	os.MkdirAll(dir, 0755)
	return dir
}

func (w *Writer) generateFilename(now time.Time, note *Note) string {
	ts := now.Format("150405")
	switch note.NoteType {
	case "image":
		return fmt.Sprintf("📷-图片-%s.md", ts)
	case "video":
		return fmt.Sprintf("🎬-视频-%s.md", ts)
	case "article":
		safeTitle := sanitizeFilename(note.Title)
		if len(safeTitle) > 30 {
			safeTitle = safeTitle[:30]
		}
		if safeTitle == "" {
			safeTitle = "文章"
		}
		return fmt.Sprintf("📄-%s-%s.md", safeTitle, ts)
	case "chat_record":
		safeTitle := sanitizeFilename(note.Title)
		if len(safeTitle) > 20 {
			safeTitle = safeTitle[:20]
		}
		if safeTitle == "" {
			safeTitle = "聊天记录"
		}
		return fmt.Sprintf("💬-%s-%s.md", safeTitle, ts)
	case "voice":
		return fmt.Sprintf("🎤-语音-%s.md", ts)
	default:
		safeTitle := sanitizeFilename(note.Content)
		if len(safeTitle) > 30 {
			safeTitle = safeTitle[:30]
		}
		if safeTitle == "" {
			safeTitle = "笔记"
		}
		return fmt.Sprintf("📝-%s-%s.md", safeTitle, ts)
	}
}

func (w *Writer) buildFrontmatter(now time.Time, note *Note) string {
	tags := []string{"微信收件箱"}
	tags = append(tags, note.Tags...)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("date: %s\n", now.Format("2006-01-02T15:04:05")))
	sb.WriteString(fmt.Sprintf("source: \"%s\"\n", note.Source))
	sb.WriteString(fmt.Sprintf("type: %s\n", note.NoteType))
	if note.URL != "" {
		sb.WriteString(fmt.Sprintf("url: \"%s\"\n", note.URL))
	}
	sb.WriteString("tags: [")
	for i, t := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(t)
	}
	sb.WriteString("]\n")
	if len(note.MediaFiles) > 0 {
		sb.WriteString("media:\n")
		for _, m := range note.MediaFiles {
			sb.WriteString(fmt.Sprintf("  - \"%s\"\n", m))
		}
	}
	sb.WriteString("---\n")
	return sb.String()
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_",
		"*", "_", "?", "_", "\"", "_",
		"<", "_", ">", "_", "|", "_",
		"\n", " ", "\r", "", "\t", " ",
	)
	return strings.TrimSpace(replacer.Replace(s))
}
