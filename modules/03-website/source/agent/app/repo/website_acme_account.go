// =============================================================================
// 模块: Website 网站管理 (agent/app/repo/website_acme_account.go)
// 文件: website_acme_account.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"gorm.io/gorm"
)

// IAcmeAccountRepo (interface)
type IAcmeAccountRepo interface {
	Page(page, size int, opts ...DBOption) (int64, []model.WebsiteAcmeAccount, error)
	GetFirst(opts ...DBOption) (*model.WebsiteAcmeAccount, error)
	Create(account model.WebsiteAcmeAccount) error
	Save(account model.WebsiteAcmeAccount) error
	DeleteBy(opts ...DBOption) error
	WithEmail(email string) DBOption
	WithType(acType string) DBOption
}

func NewIAcmeAccountRepo() IAcmeAccountRepo {
	return &WebsiteAcmeAccountRepo{}
}

type WebsiteAcmeAccountRepo struct {
}

func (w *WebsiteAcmeAccountRepo) WithEmail(email string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ?", email)
	}
}
func (w *WebsiteAcmeAccountRepo) WithType(acType string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", acType)
	}
}

func (w *WebsiteAcmeAccountRepo) Page(page, size int, opts ...DBOption) (int64, []model.WebsiteAcmeAccount, error) {
	var accounts []model.WebsiteAcmeAccount
	db := getDb(opts...).Model(&model.WebsiteAcmeAccount{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&accounts).Error
	return count, accounts, err
}

func (w *WebsiteAcmeAccountRepo) GetFirst(opts ...DBOption) (*model.WebsiteAcmeAccount, error) {
	var account model.WebsiteAcmeAccount
	db := getDb(opts...).Model(&model.WebsiteAcmeAccount{})
	if err := db.First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (w *WebsiteAcmeAccountRepo) Create(account model.WebsiteAcmeAccount) error {
	return getDb().Create(&account).Error
}

func (w *WebsiteAcmeAccountRepo) Save(account model.WebsiteAcmeAccount) error {
	return getDb().Save(&account).Error
}

func (w *WebsiteAcmeAccountRepo) DeleteBy(opts ...DBOption) error {
	return getDb(opts...).Delete(&model.WebsiteAcmeAccount{}).Error
}
