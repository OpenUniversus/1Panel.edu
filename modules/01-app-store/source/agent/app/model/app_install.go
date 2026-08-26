// =============================================================================
// 模块: App 应用商店 (agent/app/model/app_install.go)
// 文件: app_install.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"path"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
)

// AppInstall (struct)
type AppInstall struct {
	BaseModel
	Name          string `json:"name" gorm:"not null;UNIQUE"`
	AppId         uint   `json:"appId" gorm:"not null"`
	AppDetailId   uint   `json:"appDetailId" gorm:"not null"`
	Version       string `json:"version" gorm:"not null"`
	Param         string `json:"param"`
	Env           string `json:"env"`
	DockerCompose string `json:"dockerCompose" `
	Status        string `json:"status" gorm:"not null"`
	Description   string `json:"description"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName" gorm:"not null"`
	ServiceName   string `json:"serviceName" gorm:"not null"`
	HttpPort      int    `json:"httpPort"`
	HttpsPort     int    `json:"httpsPort"`
	WebUI         string `json:"webUI"`
	Favorite      bool   `json:"favorite"`
	SortOrder     int    `json:"sortOrder" gorm:"default:0"`

	App App `json:"app" gorm:"-:migration"`
}

func (i *AppInstall) GetPath() string {
	return path.Join(i.GetAppPath(), i.Name)
}

func (i *AppInstall) GetComposePath() string {
	return path.Join(i.GetAppPath(), i.Name, "docker-compose.yml")
}

func (i *AppInstall) GetEnvPath() string {
	return path.Join(i.GetAppPath(), i.Name, ".env")
}

func (i *AppInstall) GetAppPath() string {
	if i.App.Resource == constant.AppResourceLocal {
		return path.Join(global.Dir.LocalAppInstallDir, strings.TrimPrefix(i.App.Key, constant.AppResourceLocal))
	} else {
		return path.Join(global.Dir.AppInstallDir, i.App.Key)
	}
}
