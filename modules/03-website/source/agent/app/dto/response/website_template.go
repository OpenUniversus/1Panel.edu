// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/response/website_template.go)
// 文件: website_template.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
)

// WebsiteTemplateDTO (struct)
type WebsiteTemplateDTO struct {
	model.WebsiteTemplate
}

type WebsiteTemplateOutputDTO struct {
	model.WebsiteTemplateOutput
	TemplateName string `json:"templateName"`
}

type WebsitePreviewDTO struct {
	HTML string `json:"html"`
}
