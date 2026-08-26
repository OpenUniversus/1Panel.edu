// =============================================================================
// 模块: App 应用商店 (agent/app/model/app.go)
// 文件: app.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"path/filepath"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
)

// App (struct)
type App struct {
	BaseModel
	Name                 string  `json:"name" gorm:"not null"`
	Key                  string  `json:"key" gorm:"not null;"`
	ShortDescZh          string  `json:"shortDescZh" yaml:"shortDescZh"`
	ShortDescEn          string  `json:"shortDescEn" yaml:"shortDescEn"`
	Description          string  `json:"description"`
	Icon                 string  `json:"icon"`
	Type                 string  `json:"type" gorm:"not null"`
	Status               string  `json:"status" gorm:"not null"`
	Required             string  `json:"required"`
	CrossVersionUpdate   bool    `json:"crossVersionUpdate" yaml:"crossVersionUpdate"`
	Limit                int     `json:"limit" gorm:"not null"`
	Website              string  `json:"website" gorm:"not null"`
	Github               string  `json:"github" gorm:"not null"`
	Document             string  `json:"document" gorm:"not null"`
	Recommend            int     `json:"recommend" gorm:"not null"`
	Resource             string  `json:"resource" gorm:"not null;default:remote"`
	ReadMe               string  `json:"readMe"`
	LastModified         int     `json:"lastModified"`
	Architectures        string  `json:"architectures"`
	MemoryRequired       int     `json:"memoryRequired"`
	GpuSupport           bool    `json:"gpuSupport"`
	RequiredPanelVersion float64 `json:"requiredPanelVersion"`
	BatchInstallSupport  bool    `json:"batchInstallSupport" yaml:"batchInstallSupport"`

	Details []AppDetail `json:"-" gorm:"-:migration"`
	TagsKey []string    `json:"tags" yaml:"tags" gorm:"-"`
	AppTags []AppTag    `json:"-" gorm:"-:migration"`
}

func (i *App) IsLocalApp() bool {
	return i.Resource == constant.ResourceLocal
}
func (i *App) IsCustomApp() bool {
	return i.Resource == constant.AppResourceCustom
}

func (i *App) GetAppResourcePath() string {
	if i.IsLocalApp() {
		return filepath.Join(global.Dir.LocalAppResourceDir, strings.TrimPrefix(i.Key, "local"))
	}
	if i.IsCustomApp() {
		return filepath.Join(global.Dir.CustomAppResourceDir, i.Key)
	}
	return filepath.Join(global.Dir.RemoteAppResourceDir, i.Key)
}

func getLang(c *gin.Context) string {
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "en"
	}
	return lang
}

func (i *App) GetDescription(ctx *gin.Context) string {
	var translations = make(map[string]string)
	_ = json.Unmarshal([]byte(i.Description), &translations)
	lang := strings.ToLower(getLang(ctx))
	if desc, ok := translations[lang]; ok && desc != "" {
		return desc
	}
	if lang == "zh" && i.ShortDescZh != "" {
		return i.ShortDescZh
	}
	if desc, ok := translations["en"]; ok && desc != "" {
		return desc
	}
	return i.ShortDescEn
}
