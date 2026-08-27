package webassets

import "embed"

// StaticFiles 包含无需 Node 构建链的工作台资源。
//
//go:embed static/*
var StaticFiles embed.FS
