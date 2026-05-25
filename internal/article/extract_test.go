package article

import (
	"testing"
)

func TestExtract(t *testing.T) {
	url := "https://mp.weixin.qq.com/s?__biz=MzA3MTg4NjY4Mw==&mid=2457348599&idx=1&sn=c5636b343aa23ab8836c8597721a06fc"
	result, err := Extract(url)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if result.Title == "" {
		t.Error("Title is empty")
	}
	if result.Content == "" {
		t.Error("Content is empty")
	}
	t.Logf("Title: %s", result.Title)
	t.Logf("Author: %s", result.Author)
	t.Logf("Content length: %d", len(result.Content))
}
