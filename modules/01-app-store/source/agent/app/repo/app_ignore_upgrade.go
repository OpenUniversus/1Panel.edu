// =============================================================================
// 模块: App 应用商店 (agent/app/repo/app_ignore_upgrade.go)
// 文件: app_ignore_upgrade.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
	"gorm.io/gorm"
)

// AppIgnoreUpgradeRepo (struct)
type AppIgnoreUpgradeRepo struct {
}

// IAppIgnoreUpgradeRepo (interface)
type IAppIgnoreUpgradeRepo interface {
	WithScope(scope string) DBOption
	WithAppID(appID uint) DBOption
	List(opts ...DBOption) ([]model.AppIgnoreUpgrade, error)
	Create(appIgnoreUpgrade *model.AppIgnoreUpgrade) error
	Delete(opts ...DBOption) error
}

func NewIAppIgnoreUpgradeRepo() IAppIgnoreUpgradeRepo {
	return &AppIgnoreUpgradeRepo{}
}

func (a AppIgnoreUpgradeRepo) WithScope(scope string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("scope = ?", scope)
	}
}

func (a AppIgnoreUpgradeRepo) WithAppID(appID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ?", appID)
	}
}

func (a AppIgnoreUpgradeRepo) List(opts ...DBOption) ([]model.AppIgnoreUpgrade, error) {
	var appIgnoreUpgradeList []model.AppIgnoreUpgrade
	err := getDb(opts...).Find(&appIgnoreUpgradeList).Error
	return appIgnoreUpgradeList, err
}

func (a AppIgnoreUpgradeRepo) Create(appIgnoreUpgrade *model.AppIgnoreUpgrade) error {
	return global.DB.Create(appIgnoreUpgrade).Error
}

func (a AppIgnoreUpgradeRepo) Delete(opts ...DBOption) error {
	return getDb(opts...).Delete(&model.AppIgnoreUpgrade{}).Error
}
