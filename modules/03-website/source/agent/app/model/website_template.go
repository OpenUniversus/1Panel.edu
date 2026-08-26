// =============================================================================
// 模块: Website 网站管理 (agent/app/model/website_template.go)
// 文件: website_template.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// WebsiteTemplate (struct)
type WebsiteTemplate struct {
	BaseModel
	Name      string `gorm:"not null" json:"name"`
	Type      string `gorm:"not null" json:"type"` // single | multi
	Content   string `gorm:"type:longtext" json:"content"`
	FilePath  string `json:"filePath"`
	Variables string `gorm:"type:text" json:"variables"`
	Remark    string `json:"remark"`
}

func (w WebsiteTemplate) TableName() string {
	return "website_templates"
}

type WebsiteTemplateOutput struct {
	BaseModel
	Name           string `gorm:"not null" json:"name"`
	TemplateID     uint   `gorm:"not null" json:"templateID"`
	TemplateType   string `json:"templateType"`
	VariableValues string `gorm:"type:text" json:"variableValues"`
	OutputPath     string `json:"outputPath"`
}

func (w WebsiteTemplateOutput) TableName() string {
	return "website_template_outputs"
}
