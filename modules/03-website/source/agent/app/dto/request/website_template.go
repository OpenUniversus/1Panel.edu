// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/request/website_template.go)
// 文件: website_template.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package request

import (
	"github.com/1Panel-dev/1Panel/agent/app/dto"
)

// WebsiteTemplateSearch (struct)
type WebsiteTemplateSearch struct {
	dto.PageInfo
	Name string `json:"name"`
	Type string `json:"type"`
}

type WebsiteTemplateCreate struct {
	Name      string `json:"name" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=single multi"`
	Content   string `json:"content"`
	FilePath  string `json:"filePath"`
	Variables string `json:"variables"`
	Remark    string `json:"remark"`
}

type WebsiteTemplateUpdate struct {
	ID        uint   `json:"id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=single multi"`
	Content   string `json:"content"`
	FilePath  string `json:"filePath"`
	Variables string `json:"variables"`
	Remark    string `json:"remark"`
}

type WebsiteTemplateOutputSearch struct {
	dto.PageInfo
	TemplateID uint `json:"templateID"`
}

type WebsiteTemplateOutputCreate struct {
	TemplateID     uint              `json:"templateID" validate:"required"`
	Name           string            `json:"name" validate:"required"`
	VariableValues map[string]string `json:"variableValues"`
}

// WebsitePreviewReq (struct)
type WebsitePreviewReq struct {
	TemplateID     uint              `json:"templateID" validate:"required"`
	VariableValues map[string]string `json:"variableValues"`
}
