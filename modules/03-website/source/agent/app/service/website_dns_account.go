// =============================================================================
// 模块: Website 网站管理 (agent/app/service/website_dns_account.go)
// 文件: website_dns_account.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"encoding/json"

	"github.com/1Panel-dev/1Panel/agent/app/repo"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/dto/response"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/buserr"
)

// WebsiteDnsAccountService (struct)
type WebsiteDnsAccountService struct {
}

// IWebsiteDnsAccountService (interface)
type IWebsiteDnsAccountService interface {
	Page(search dto.PageInfo) (int64, []response.WebsiteDnsAccountDTO, error)
	Create(create request.WebsiteDnsAccountCreate) (request.WebsiteDnsAccountCreate, error)
	Update(update request.WebsiteDnsAccountUpdate) (request.WebsiteDnsAccountUpdate, error)
	Delete(id uint) error
}

func NewIWebsiteDnsAccountService() IWebsiteDnsAccountService {
	return &WebsiteDnsAccountService{}
}

func (w WebsiteDnsAccountService) Page(search dto.PageInfo) (int64, []response.WebsiteDnsAccountDTO, error) {
	total, accounts, err := websiteDnsRepo.Page(search.Page, search.PageSize, repo.WithOrderDesc("created_at"))
	var accountDTOs []response.WebsiteDnsAccountDTO
	for _, account := range accounts {
		auth := make(map[string]string)
		_ = json.Unmarshal([]byte(account.Authorization), &auth)
		accountDTOs = append(accountDTOs, response.WebsiteDnsAccountDTO{
			WebsiteDnsAccount: account,
			Authorization:     auth,
		})
	}
	return total, accountDTOs, err
}

func (w WebsiteDnsAccountService) Create(create request.WebsiteDnsAccountCreate) (request.WebsiteDnsAccountCreate, error) {
	exist, _ := websiteDnsRepo.GetFirst(repo.WithByName(create.Name))
	if exist != nil {
		return request.WebsiteDnsAccountCreate{}, buserr.New("ErrNameIsExist")
	}

	authorization, err := json.Marshal(create.Authorization)
	if err != nil {
		return request.WebsiteDnsAccountCreate{}, err
	}

	if err := websiteDnsRepo.Create(model.WebsiteDnsAccount{
		Name:          create.Name,
		Type:          create.Type,
		Authorization: string(authorization),
	}); err != nil {
		return request.WebsiteDnsAccountCreate{}, err
	}

	return create, nil
}

func (w WebsiteDnsAccountService) Update(update request.WebsiteDnsAccountUpdate) (request.WebsiteDnsAccountUpdate, error) {
	authorization, err := json.Marshal(update.Authorization)
	if err != nil {
		return request.WebsiteDnsAccountUpdate{}, err
	}
	exists, _ := websiteDnsRepo.List(repo.WithByName(update.Name))
	for _, exist := range exists {
		if exist.ID != update.ID {
			return request.WebsiteDnsAccountUpdate{}, buserr.New("ErrNameIsExist")
		}
	}
	if err := websiteDnsRepo.Save(model.WebsiteDnsAccount{
		BaseModel: model.BaseModel{
			ID: update.ID,
		},
		Name:          update.Name,
		Type:          update.Type,
		Authorization: string(authorization),
	}); err != nil {
		return request.WebsiteDnsAccountUpdate{}, err
	}

	return update, nil
}

func (w WebsiteDnsAccountService) Delete(id uint) error {
	if ssls, _ := websiteSSLRepo.List(websiteSSLRepo.WithByDnsAccountId(id)); len(ssls) > 0 {
		return buserr.New("ErrAccountCannotDelete")
	}
	return websiteDnsRepo.DeleteBy(repo.WithByID(id))
}
