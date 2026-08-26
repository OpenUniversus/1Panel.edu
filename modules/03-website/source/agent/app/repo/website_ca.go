// =============================================================================
// 模块: Website 网站管理 (agent/app/repo/website_ca.go)
// 文件: website_ca.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"context"

	"github.com/1Panel-dev/1Panel/agent/app/model"
)

// WebsiteCARepo (struct)
type WebsiteCARepo struct {
}

func NewIWebsiteCARepo() IWebsiteCARepo {
	return &WebsiteCARepo{}
}

// IWebsiteCARepo (interface)
type IWebsiteCARepo interface {
	Page(page, size int, opts ...DBOption) (int64, []model.WebsiteCA, error)
	GetFirst(opts ...DBOption) (model.WebsiteCA, error)
	List(opts ...DBOption) ([]model.WebsiteCA, error)
	Create(ctx context.Context, ca *model.WebsiteCA) error
	DeleteBy(opts ...DBOption) error
}

func (w WebsiteCARepo) Page(page, size int, opts ...DBOption) (int64, []model.WebsiteCA, error) {
	var caList []model.WebsiteCA
	db := getDb(opts...).Model(&model.WebsiteCA{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&caList).Error
	return count, caList, err
}

func (w WebsiteCARepo) GetFirst(opts ...DBOption) (model.WebsiteCA, error) {
	var ca model.WebsiteCA
	db := getDb(opts...).Model(&model.WebsiteCA{})
	if err := db.First(&ca).Error; err != nil {
		return ca, err
	}
	return ca, nil
}

func (w WebsiteCARepo) List(opts ...DBOption) ([]model.WebsiteCA, error) {
	var caList []model.WebsiteCA
	db := getDb(opts...).Model(&model.WebsiteCA{})
	err := db.Find(&caList).Error
	return caList, err
}

func (w WebsiteCARepo) Create(ctx context.Context, ca *model.WebsiteCA) error {
	return getTx(ctx).Create(ca).Error
}

func (w WebsiteCARepo) DeleteBy(opts ...DBOption) error {
	return getDb(opts...).Delete(&model.WebsiteCA{}).Error
}
