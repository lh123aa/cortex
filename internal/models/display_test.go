package models

import (
	"strings"
	"testing"
)

func TestChunkDisplayTextPrefersOriginal(t *testing.T) {
	ch := &Chunk{
		Content:    "File: /docs/a.md | section\n\nGo 标准库与并发模型",
		ContentRaw: "Go 标 准 库 与 并 发 模 型",
	}
	got := ch.DisplayText()
	if strings.Contains(got, "标 准") {
		t.Fatalf("DisplayText returned FTS-tokenized text: %q", got)
	}
	if !strings.Contains(got, "标准库与并发模型") {
		t.Fatalf("DisplayText should return original text, got %q", got)
	}
}

func TestChunkDisplayTextStripsBreadcrumb(t *testing.T) {
	ch := &Chunk{
		Content:    "Section: 部署 > 单二进制\n\n一条命令启动所有服务。",
		ContentRaw: "一 条 命 令 启 动 所 有 服 务 。",
	}
	got := ch.DisplayText()
	if strings.HasPrefix(got, "Section:") {
		t.Fatalf("breadcrumb prefix should be stripped for display, got %q", got)
	}
	if !strings.Contains(got, "一条命令启动所有服务") {
		t.Fatalf("body lost: %q", got)
	}
}

func TestChunkDisplayTextFallsBackToRaw(t *testing.T) {
	ch := &Chunk{Content: "", ContentRaw: "fallback body"}
	if got := ch.DisplayText(); got != "fallback body" {
		t.Fatalf("expected fallback to ContentRaw, got %q", got)
	}
}

func TestNilChunkDisplayText(t *testing.T) {
	var ch *Chunk
	if got := ch.DisplayText(); got != "" {
		t.Fatalf("nil chunk should return empty string, got %q", got)
	}
}

func TestStripChunkBreadcrumbKeepsPlainBody(t *testing.T) {
	body := "这是没有前缀标记的普通正文。\n\n第二段。"
	if got := stripChunkBreadcrumb(body); got != body {
		t.Fatalf("plain body should be untouched, got %q", got)
	}
}
