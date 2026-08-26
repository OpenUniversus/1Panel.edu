// =============================================================================
// 模块: App 应用商店 (agent/app/service/app_ingore_upgrade.go)
// 文件: app_ingore_upgrade.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/dto/response"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// AppIgnoreUpgradeService (struct)
type AppIgnoreUpgradeService struct {
}

// IAppIgnoreUpgradeService (interface)
type IAppIgnoreUpgradeService interface {
	List() ([]response.AppIgnoreUpgradeDTO, error)
	CreateAppIgnore(req request.AppIgnoreUpgradeReq) error
	Delete(req request.ReqWithID) error
}

func NewIAppIgnoreUpgradeService() IAppIgnoreUpgradeService {
	return AppIgnoreUpgradeService{}
}

func (a AppIgnoreUpgradeService) List() ([]response.AppIgnoreUpgradeDTO, error) {
	var res []response.AppIgnoreUpgradeDTO
	ignores, err := appIgnoreUpgradeRepo.List()
	if err != nil {
		return nil, err
	}
	for _, ignore := range ignores {
		dto := response.AppIgnoreUpgradeDTO{
			ID:          ignore.ID,
			AppID:       ignore.AppID,
			AppDetailID: ignore.AppDetailID,
			Scope:       ignore.Scope,
		}
		app, err := appRepo.GetFirst(repo.WithByID(ignore.AppID))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = appIgnoreUpgradeRepo.Delete(repo.WithByID(ignore.ID))
			continue
		}
		dto.Name = app.Name
		if ignore.Scope == "version" {
			appDetail, err := appDetailRepo.GetFirst(repo.WithByID(ignore.AppDetailID))
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = appIgnoreUpgradeRepo.Delete(repo.WithByID(ignore.ID))
				continue
			}
			dto.Version = appDetail.Version
		}
		res = append(res, dto)
	}
	return res, nil
}

func (a AppIgnoreUpgradeService) CreateAppIgnore(req request.AppIgnoreUpgradeReq) error {
	appIgnoreUpgrade := model.AppIgnoreUpgrade{
		AppID: req.AppID,
		Scope: req.Scope,
	}
	if req.Scope == "version" {
		appIgnoreUpgrade.AppDetailID = req.AppDetailID
	}
	if req.Scope == "all" {
		_ = appIgnoreUpgradeRepo.Delete(appInstallRepo.WithAppId(req.AppID))
	}
	if err := appIgnoreUpgradeRepo.Create(&appIgnoreUpgrade); err != nil {
		return err
	}
	return nil
}

func (a AppIgnoreUpgradeService) Delete(req request.ReqWithID) error {
	return appIgnoreUpgradeRepo.Delete(repo.WithByID(req.ID))
}
