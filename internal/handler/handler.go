package handler

import (
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eatmoreapple/openwechat"

	"wechat-obsidian-bot/internal/article"
	"wechat-obsidian-bot/internal/media"
	"wechat-obsidian-bot/internal/writer"
)

type Handler struct {
	writer *writer.Writer
	cfg    Config
}

type Config struct {
	DownloadMedia   bool
	ExtractArticles bool
	AutoFriendAdd   bool
	WelcomeMsg      string
}

func New(w *writer.Writer, cfg Config) *Handler {
	return &Handler{writer: w, cfg: cfg}
}

func (h *Handler) Handle(msg *openwechat.Message) {
	if msg.IsNotify() || msg.MsgType == 51 {
		return
	}
	if msg.IsSystem() {
		return
	}

	log.Printf("[MSG] type=%d appType=%d text=%v pic=%v media=%v friend=%v content=%d",
		msg.MsgType, msg.AppMsgType,
		msg.IsText(), msg.IsPicture(), msg.IsMedia(),
		msg.IsSendByFriend(), len(msg.Content))

	if msg.IsFriendAdd() {
		h.handleFriendAdd(msg)
		return
	}

	sender, err := msg.Sender()
	if err != nil {
		log.Printf("[WARN] sender lookup failed: %v", err)
		return
	}

	senderName := senderName(sender)
	if !msg.IsSendByFriend() && !msg.IsSendBySelf() {
		return
	}

	switch {
	case msg.IsText():
		h.handleText(msg, senderName)
	case msg.IsPicture():
		h.handleImage(msg, senderName)
	case msg.IsVideo():
		h.handleVideo(msg, senderName)
	case msg.IsVoice():
		h.handleVoice(msg, senderName)
	case msg.IsCard():
		h.handleCard(msg, senderName)
	case msg.IsMedia():
		h.handleAppMessage(msg, senderName)
	case msg.IsRecalled():
		log.Printf("[%s] 撤回了一条消息", senderName)
	default:
		h.handleGeneric(msg, senderName)
	}

	msg.AsRead()
}

// ---- Text ----

func (h *Handler) handleText(msg *openwechat.Message, sender string) {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return
	}

	if article.IsArticleURL(content) {
		h.processArticle("", content, sender, msg)
		return
	}

	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		note := &writer.Note{
			Title:    truncateText(content, 50),
			Content:  content,
			Source:   sender,
			NoteType: "link",
			URL:      content,
			Tags:     []string{"链接"},
		}
		if _, err := h.writer.WriteNote(note); err != nil {
			log.Printf("[ERR] write link: %v", err)
			return
		}
		msg.ReplyText("链接已保存 ✅")
		log.Printf("[%s] link saved", sender)
		return
	}

	note := &writer.Note{
		Content:  content,
		Source:   sender,
		NoteType: "text",
		Tags:     []string{"闪念笔记"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write text: %v", err)
		return
	}
	msg.ReplyText("已保存 ✅")
	log.Printf("[%s] text saved", sender)
}

// ---- Image ----

func (h *Handler) handleImage(msg *openwechat.Message, sender string) {
	imgName := msg.FileName
	if imgName == "" {
		imgName = fmt.Sprintf("img_%d.jpg", time.Now().UnixNano()/1e6)
	}
	destDir := h.writer.AttachmentsDir()
	imgPath := filepath.Join(destDir, imgName)

	if err := msg.SaveFileToLocal(imgPath); err != nil {
		log.Printf("[ERR] save image: %v", err)
		msg.ReplyText("保存图片失败: " + err.Error())
		return
	}

	relPath := "attachments/" + imgName
	note := &writer.Note{
		Title:      sender + " 发来一张图片",
		Content:    fmt.Sprintf("![[%s]]\n\n来自：%s", relPath, sender),
		Source:     sender,
		NoteType:   "image",
		MediaFiles: []string{relPath},
		Tags:       []string{"图片"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write image note: %v", err)
		return
	}
	msg.ReplyText("图片已保存 ✅")
	log.Printf("[%s] image saved", sender)
}

// ---- Video ----

func (h *Handler) handleVideo(msg *openwechat.Message, sender string) {
	videoName := msg.FileName
	if videoName == "" {
		videoName = fmt.Sprintf("video_%d.mp4", time.Now().UnixNano()/1e6)
	}
	destDir := h.writer.AttachmentsDir()
	videoPath := filepath.Join(destDir, videoName)

	if err := msg.SaveFileToLocal(videoPath); err != nil {
		log.Printf("[ERR] save video: %v", err)
		msg.ReplyText("保存视频失败: " + err.Error())
		return
	}

	relPath := "attachments/" + videoName
	note := &writer.Note{
		Title:      sender + " 发来一个视频",
		Content:    fmt.Sprintf("![[%s]]\n\n来自：%s", relPath, sender),
		Source:     sender,
		NoteType:   "video",
		MediaFiles: []string{relPath},
		Tags:       []string{"视频"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write video note: %v", err)
		return
	}
	msg.ReplyText("视频已保存 ✅")
	log.Printf("[%s] video saved", sender)
}

// ---- Voice ----

func (h *Handler) handleVoice(msg *openwechat.Message, sender string) {
	voiceName := msg.FileName
	if voiceName == "" {
		voiceName = fmt.Sprintf("voice_%d.silk", time.Now().UnixNano()/1e6)
	}
	destDir := h.writer.AttachmentsDir()
	voicePath := filepath.Join(destDir, voiceName)

	if err := msg.SaveFileToLocal(voicePath); err != nil {
		log.Printf("[ERR] save voice: %v", err)
		msg.ReplyText("保存语音失败: " + err.Error())
		return
	}

	relPath := "attachments/" + voiceName
	note := &writer.Note{
		Title:      sender + " 发来一段语音",
		Content:    fmt.Sprintf("语音文件：`%s`\n\n> 🔄 正在转录中...\n\n来自：%s", relPath, sender),
		Source:     sender,
		NoteType:   "voice",
		MediaFiles: []string{relPath},
		Tags:       []string{"语音"},
	}
	filePath, err := h.writer.WriteNote(note)
	if err != nil {
		log.Printf("[ERR] write voice note: %v", err)
		return
	}

	msg.ReplyText("语音已保存，正在转录... 🎤➡️📝")

	// Transcribe asynchronously
	go h.transcribeVoice(voicePath, filePath, sender, msg)
	log.Printf("[%s] voice saved (transcribing...)", sender)
}

func (h *Handler) transcribeVoice(voicePath, notePath, sender string, msg *openwechat.Message) {
	result, err := article.TranscribeVoice(voicePath)
	if err != nil {
		log.Printf("[ERR] voice transcription: %v", err)
		msg.ReplyText("语音转录失败: " + err.Error())
		return
	}

	if result.Error != "" {
		log.Printf("[ERR] voice transcription: %s", result.Error)
		msg.ReplyText("语音转录失败: " + result.Error)
		return
	}

	if result.Text == "" {
		msg.ReplyText("语音转录完成，但未识别到内容")
		return
	}

	// Update the note with transcription
	content := fmt.Sprintf("语音文件：`%s`\n\n## 转录内容\n\n%s\n\n来自：%s",
		"attachments/"+filepath.Base(voicePath),
		result.Text, sender)

	if err := h.writer.UpdateNoteContent(notePath, content); err != nil {
		log.Printf("[ERR] update voice note: %v", err)
		return
	}
	log.Printf("[%s] voice transcribed: %s", sender, truncateText(result.Text, 50))
	msg.ReplyText(fmt.Sprintf("语音转录完成 ✅\n> %s", truncateText(result.Text, 60)))
}

// ---- Card ----

func (h *Handler) handleCard(msg *openwechat.Message, sender string) {
	note := &writer.Note{
		Title:    sender + " 分享了一张名片",
		Content:  fmt.Sprintf("```\n%s\n```\n\n来自：%s", msg.Content, sender),
		Source:   sender,
		NoteType: "card",
		Tags:     []string{"名片"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write card: %v", err)
		return
	}
	msg.ReplyText("名片已保存 ✅")
	log.Printf("[%s] card saved", sender)
}

// ---- AppMessage (articles, chat records, links) ----

func (h *Handler) handleAppMessage(msg *openwechat.Message, sender string) {
	var appMsg AppMessageXML
	if err := xml.Unmarshal([]byte(msg.Content), &appMsg); err != nil {
		log.Printf("[WARN] XML parse failed, saving raw")
		h.saveRaw("appmsg", msg.Content, sender)
		return
	}

	app := appMsg.AppMsg
	title := app.Title
	url := app.URL

	switch {
	case msg.AppMsgType == openwechat.AppMsgTypeUrl && article.IsArticleURL(url):
		h.processArticle(title, url, sender, msg)

	case msg.AppMsgType == openwechat.AppMsgTypeUrl && url != "":
		h.saveLink(title, url, app.Des, sender, msg)

	case msg.AppMsgType == openwechat.AppMsgTypeAttach:
		if app.RecordItem != nil {
			h.processChatRecord(&appMsg, sender, msg)
		} else if h.isDocumentFile(msg) {
			h.handleFileAttachment(msg, sender, title, app.Des)
		} else {
			h.saveLink(title, url, app.Des, sender, msg)
		}

	case url != "":
		if article.IsArticleURL(url) {
			h.processArticle(title, url, sender, msg)
		} else {
			h.saveLink(title, url, app.Des, sender, msg)
		}

	case msg.AppMsgType == 51:
			// Video channel / note content
			h.processUnsupportedAppMsg(&appMsg, sender, msg)

		default:
			if app.RecordItem != nil || strings.Contains(msg.Content, "<recorditem>") {
				h.processChatRecord(&appMsg, sender, msg)
			} else {
				// Save raw XML for unknown app types (mini programs, video channel, etc.)
				h.saveLink(title, url, app.Des, sender, msg)
			}
		}
	}

// processUnsupportedAppMsg handles content that the desktop WeChat client cannot render.
func (h *Handler) processUnsupportedAppMsg(appMsg *AppMessageXML, sender string, msg *openwechat.Message) {
	app := appMsg.AppMsg
	title := app.Title

	isVideoChannel := strings.Contains(title, "version does not support") ||
		strings.Contains(title, "微信") ||
		strings.Contains(msg.Content, "finder") || strings.Contains(msg.Content, "Finder")

	var content strings.Builder
	if isVideoChannel {
		content.WriteString("**视频号内容** — 桌面版微信无法渲染\n\n")
		content.WriteString("> 请在手机上打开微信查看完整内容\n\n")
	} else {
		content.WriteString(fmt.Sprintf("**%s** — 桌面版微信不支持此内容\n\n", title))
		content.WriteString("> 请更新微信或使用手机查看\n\n")
	}

	if app.URL != "" && !strings.Contains(app.URL, "support.weixin.qq.com/update") {
		content.WriteString(fmt.Sprintf("原始链接：[%s](%s)\n\n", app.URL, app.URL))
	}

	content.WriteString(
		fmt.Sprintf("来自：%s\n\n---\n\n<details>\n<summary>原始 XML</summary>\n\n```xml\n%s\n```\n</details>",
			sender, msg.Content))

	note := &writer.Note{
		Title:    "视频号: " + sender,
		Content:  content.String(),
		Source:   sender,
		NoteType: "video_channel",
		Tags:     []string{"视频号"},
	}

	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write video channel note: %v", err)
		return
	}
	msg.ReplyText("视频号内容已保存\n> 桌面版微信无法渲染，请在手机上查看")
	log.Printf("[%s] video channel saved", sender)
}

// ---- Article extraction// ---- Article extraction (async, with image download and feedback) ----

func (h *Handler) processArticle(title, url, sender string, msg *openwechat.Message) {
	if title == "" {
		title = url
	}

	if h.cfg.ExtractArticles {
		msg.ReplyText("文章提取中，请稍候... ⏳")

		go func() {
			result, err := article.Extract(url)
			if err != nil {
				log.Printf("[WARN] article extract failed: %v", err)
				h.saveArticleFallback(title, url, sender)
				msg.ReplyText(fmt.Sprintf("文章提取失败，已保存链接 ⚠️\n> %s", truncateText(title, 30)))
				return
			}

			// Extract embedded video/audio links before image replacement
			mediaLinks := h.extractMediaLinks(result.Content)

			// Download images: from regex in content + from feedgrab's all_images list
			contentWithLocalImages := h.downloadArticleImages(result.Content, result.AllImages)

			// Append media links section if any found
			if len(mediaLinks) > 0 {
				contentWithLocalImages += "\n\n---\n\n## 📺 原文中的媒体链接\n\n"
				contentWithLocalImages += "> 以下链接来自原文中嵌入的视频/音频，需跳转至外部平台观看\n\n"
				for _, link := range mediaLinks {
					contentWithLocalImages += fmt.Sprintf("- [🔗 外部链接](%s)\n", link)
				}
			}

			articleNote := &writer.Note{
				Title:    result.Title,
				Content:  contentWithLocalImages,
				Source:   sender,
				NoteType: "article",
				URL:      url,
				Tags:     []string{"文章"},
			}
			if result.Author != "" {
				articleNote.Tags = append(articleNote.Tags, result.Author)
			}

			filePath, err := h.writer.WriteNote(articleNote)
			if err != nil {
				log.Printf("[ERR] write article: %v", err)
				msg.ReplyText("文章保存失败 ⚠️")
				return
			}
			log.Printf("[%s] article saved: %s → %s", sender, result.Title, filepath.Base(filePath))
			msg.ReplyText(fmt.Sprintf("文章已保存到 Obsidian ✅\n📄 %s", truncateText(result.Title, 40)))
		}()
		return
	}

	h.saveArticleFallback(title, url, sender)
	msg.ReplyText("文章链接已保存 ✅")
}

// downloadArticleImages finds image URLs in markdown content, downloads them,
// and replaces remote URLs with local Obsidian embed syntax.
func (h *Handler) downloadArticleImages(content string, extraImages []string) string {
	destDir := h.writer.AttachmentsDir()
	replacements := make(map[string]string) // original → replacement

	// Collect matches from both markdown and HTML image patterns
	type imgMatch struct{ full, url string }
	var allMatches []imgMatch

	for _, m := range imageMarkdownRegex.FindAllStringSubmatch(content, -1) {
		allMatches = append(allMatches, imgMatch{m[0], m[2]})
	}
	for _, m := range htmlImgRegex.FindAllStringSubmatch(content, -1) {
		allMatches = append(allMatches, imgMatch{m[0], m[1]})
	}

	// Also process images from feedgrab's all_images list (captures table images
	// that feedgrab's HTML->Markdown converter stripped). These get appended at end.
	type extraImg struct{ url, localRel string }
	var extraDownloaded []extraImg

	for _, imgURL := range extraImages {
		// Skip if already in our match list
		dup := false
		for _, m := range allMatches {
			if m.url == imgURL {
				dup = true
				break
			}
		}
		if dup {
			continue
		}

		filename := fmt.Sprintf("article_%x", hashStr(imgURL))
		localPath, err := media.DownloadFromURL(imgURL, destDir, filename)
		if err != nil {
			log.Printf("[WARN] download extra image failed: %s → %v", truncateText(imgURL, 60), err)
			continue
		}
		relPath := "attachments/" + filepath.Base(localPath)
		extraDownloaded = append(extraDownloaded, extraImg{imgURL, relPath})
		log.Printf("[IMG] downloaded extra: %s → %s", truncateText(imgURL, 60), filepath.Base(localPath))
	}

	if len(allMatches) == 0 && len(extraDownloaded) == 0 {
		return content
	}

	for _, m := range allMatches {
		fullMatch := m.full
		imgURL := m.url

		// Generate filename (extension auto-detected by DownloadFromURL)
		filename := fmt.Sprintf("article_%x", hashStr(imgURL))

		localPath, err := media.DownloadFromURL(imgURL, destDir, filename)
		if err != nil {
			log.Printf("[WARN] download article image failed: %s → %v", truncateText(imgURL, 60), err)
			continue
		}

		relPath := "attachments/" + filepath.Base(localPath)
		replacements[fullMatch] = fmt.Sprintf("![图片](%s)", relPath)
		log.Printf("[IMG] downloaded: %s → %s", truncateText(imgURL, 60), filepath.Base(localPath))
	}

	// Apply replacements
	result := content
	for orig, repl := range replacements {
		result = strings.Replace(result, orig, repl, 1)
	}

	// Append extra images (from feedgrab's all_images list) at the end
	if len(extraDownloaded) > 0 {
		result += "\n\n---\n\n## 🖼️ 补充图片\n\n"
		result += "> 以下图片来自原文但未在正文中显示（可能在表格内或被格式转换丢弃）\n\n"
		for _, x := range extraDownloaded {
			result += fmt.Sprintf("![图片](%s)\n\n", x.localRel)
		}
	}

	return result
}

func (h *Handler) saveArticleFallback(title, url, sender string) {
	if title == "" || title == url {
		title = "公众号文章"
	}
	note := &writer.Note{
		Title:    title,
		Content:  fmt.Sprintf("[%s](%s)\n\n来自：%s\n\n> ⚠️ 正文提取失败，仅保存了链接", title, url, sender),
		Source:   sender,
		NoteType: "article_link",
		URL:      url,
		Tags:     []string{"文章", "待提取"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write article fallback: %v", err)
	}
}

// ---- Chat record forwarding ----

func (h *Handler) processChatRecord(appMsg *AppMessageXML, sender string, msg *openwechat.Message) {
	app := appMsg.AppMsg
	title := app.Title

	var content strings.Builder
	content.WriteString(fmt.Sprintf("转发自：%s\n\n---\n\n", sender))

	if app.RecordItem != nil {
		lines := strings.Split(strings.TrimSpace(app.RecordItem.Desc), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				name := strings.TrimSpace(line[:idx])
				text := strings.TrimSpace(line[idx+1:])
				content.WriteString(fmt.Sprintf("**%s**：%s\n\n", name, text))
			} else {
				content.WriteString(fmt.Sprintf("%s\n\n", line))
			}
		}
	}
	if app.Des != "" && (app.RecordItem == nil || app.Des != app.RecordItem.Desc) {
		content.WriteString(fmt.Sprintf("\n> %s\n", app.Des))
	}

	note := &writer.Note{
		Title:    title,
		Content:  content.String(),
		Source:   sender,
		NoteType: "chat_record",
		Tags:     []string{"聊天记录"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write chat record: %v", err)
		return
	}
	msg.ReplyText("聊天记录已保存 ✅")
	log.Printf("[%s] chat record saved: %s", sender, title)
}

// ---- Helpers ----

func (h *Handler) saveLink(title, url, desc, sender string, msg *openwechat.Message) {
	if title == "" {
		title = url
	}
	note := &writer.Note{
		Title:    truncateText(title, 50),
		Content:  fmt.Sprintf("[%s](%s)\n\n%s\n\n来自：%s", title, url, desc, sender),
		Source:   sender,
		NoteType: "link",
		URL:      url,
		Tags:     []string{"链接"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write link: %v", err)
		return
	}
	msg.ReplyText("链接已保存 ✅")
	log.Printf("[%s] link saved", sender)
}

// ---- File attachments (PDF, Word, Excel, PPT, etc.) ----

func (h *Handler) isDocumentFile(msg *openwechat.Message) bool {
	if !msg.HasFile() {
		return false
	}
	name := strings.ToLower(msg.FileName)
	for _, ext := range []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".ppt", ".pptx", ".txt", ".csv", ".zip",
		".rar", ".7z", ".md", ".json", ".xml",
		".html", ".htm", ".epub", ".mobi",
	} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func (h *Handler) handleFileAttachment(msg *openwechat.Message, sender, title, desc string) {
	fileName := msg.FileName
	destDir := h.writer.AttachmentsDir()
	filePath := filepath.Join(destDir, fileName)

	if err := msg.SaveFileToLocal(filePath); err != nil {
		log.Printf("[ERR] save file: %v", err)
		msg.ReplyText("保存文件失败: " + err.Error())
		return
	}

	relPath := "attachments/" + fileName
	displayTitle := title
	if displayTitle == "" {
		displayTitle = fileName
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("[[%s|📂 %s]]\n\n", relPath, fileName))
	if desc != "" {
		content.WriteString(fmt.Sprintf("> %s\n\n", desc))
	}
	content.WriteString(fmt.Sprintf("来自：%s", sender))

	note := &writer.Note{
		Title:      "📎 " + displayTitle,
		Content:    content.String(),
		Source:     sender,
		NoteType:   "file",
		MediaFiles: []string{relPath},
		Tags:       []string{"文件"},
	}

	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write file note: %v", err)
		return
	}
	msg.ReplyText(fmt.Sprintf("文件已保存 ✅\n📎 %s", fileName))
	log.Printf("[%s] file saved: %s", sender, fileName)
}

func (h *Handler) handleGeneric(msg *openwechat.Message, sender string) {
	h.saveRaw("unknown", msg.Content, sender)
}

func (h *Handler) saveRaw(kind, content, sender string) {
	note := &writer.Note{
		Title:    fmt.Sprintf("未识别消息 (%s)", kind),
		Content:  fmt.Sprintf("```xml\n%s\n```\n\n来自：%s", content, sender),
		Source:   sender,
		NoteType: kind,
		Tags:     []string{"未识别"},
	}
	if _, err := h.writer.WriteNote(note); err != nil {
		log.Printf("[ERR] write raw: %v", err)
	}
	log.Printf("[%s] raw %s message saved", sender, kind)
}

func (h *Handler) handleFriendAdd(msg *openwechat.Message) {
	if !h.cfg.AutoFriendAdd {
		log.Println("好友申请: 自动通过已关闭")
		return
	}
	if _, err := msg.Agree(h.cfg.WelcomeMsg); err != nil {
		log.Printf("[ERR] accept friend: %v", err)
		return
	}
	log.Println("已通过好友申请")
}

func senderName(sender *openwechat.User) string {
	if sender == nil {
		return "未知"
	}
	if sender.RemarkName != "" {
		return sender.RemarkName
	}
	if sender.NickName != "" {
		return sender.NickName
	}
	return sender.UserName
}

func truncateText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// ---- Helpers ----

var imageMarkdownRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
var htmlImgRegex = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
var iframeRegex = regexp.MustCompile(`<iframe[^>]+src=["']([^"']+)["']`)
var videoSrcRegex = regexp.MustCompile(`<video[^>]+src=["']([^"']+)["']`)
var sourceSrcRegex = regexp.MustCompile(`<source[^>]+src=["']([^"']+)["']`)
var mdLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
var rawURLRegex = regexp.MustCompile(`https?://[^\s<>"')\]]+`)

// extractMediaLinks scans article content for embedded video/audio URLs.
func (h *Handler) extractMediaLinks(content string) []string {
	seen := make(map[string]bool)
	var links []string

	for _, m := range iframeRegex.FindAllStringSubmatch(content, -1) {
		url := strings.TrimSpace(m[1])
		if url != "" && !seen[url] {
			seen[url] = true
			links = append(links, url)
		}
	}

	for _, re := range []*regexp.Regexp{videoSrcRegex, sourceSrcRegex} {
		for _, m := range re.FindAllStringSubmatch(content, -1) {
			url := strings.TrimSpace(m[1])
			if url != "" && !seen[url] {
				seen[url] = true
				links = append(links, url)
			}
		}
	}

	for _, m := range mdLinkRegex.FindAllStringSubmatch(content, -1) {
		url := strings.TrimSpace(m[2])
		if isMediaPlatformURL(url) && !seen[url] {
			seen[url] = true
			links = append(links, url)
		}
	}

	for _, m := range rawURLRegex.FindAllStringSubmatch(content, -1) {
		url := strings.TrimSpace(m[0])
		if isMediaPlatformURL(url) && !seen[url] {
			seen[url] = true
			links = append(links, url)
		}
	}

	return links
}

func isMediaPlatformURL(url string) bool {
	lower := strings.ToLower(url)
	for _, kw := range []string{
		"v.qq.com", "video.qq.com",
		"bilibili.com/video", "b23.tv",
		"youtube.com/watch", "youtu.be/",
		"youku.com/v_show", "v.youku.com",
		"ixigua.com", "douyin.com/video",
		"xhslink.com",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func hashStr(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// ---- XML structures ----

type AppMessageXML struct {
	XMLName xml.Name    `xml:"msg"`
	AppMsg  AppMsgField `xml:"appmsg"`
}

type AppMsgField struct {
	Title      string         `xml:"title"`
	Des        string         `xml:"des"`
	Type       string         `xml:"type"`
	URL        string         `xml:"url"`
	AppID      string         `xml:"appid"`
	SDKVer     string         `xml:"sdkver"`
	RecordItem *RecordItemXML `xml:"recorditem"`
	AppInfo    *AppInfoXML    `xml:"appinfo"`
}

type RecordItemXML struct {
	Title string `xml:"title"`
	Desc  string `xml:"desc"`
}

type AppInfoXML struct {
	Version string `xml:"version"`
	AppName string `xml:"appname"`
}
