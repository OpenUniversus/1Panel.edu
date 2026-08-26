// =============================================================================
// 模块: Website 网站管理 (agent/app/service/website_op.go)
// 文件: website_op.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"context"
	"fmt"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/app/task"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"sort"
)

func (w WebsiteService) BatchOpWebsite(req request.BatchWebsiteOp) error {
	websites, _ := websiteRepo.List(repo.WithByIDs(req.IDs))
	opTask, err := task.NewTaskWithOps(i18n.GetMsgByKey("Status"), task.TaskBatch, task.TaskScopeWebsite, req.TaskID, 0)
	if err != nil {
		return err
	}
	sort.SliceStable(websites, func(i, j int) bool {
		if websites[i].Type == constant.Subsite && websites[j].Type != constant.Subsite {
			return true
		}
		if websites[i].Type != constant.Subsite && websites[j].Type == constant.Subsite {
			return false
		}
		return false
	})
	opWebsiteTask := func(t *task.Task) error {
		for _, web := range websites {
			msg := fmt.Sprintf("%s %s", i18n.GetMsgByKey(req.Operate), web.PrimaryDomain)
			switch req.Operate {
			case constant.StopWeb, constant.StartWeb:
				if err := opWebsite(&web, req.Operate); err != nil {
					t.LogFailedWithErr(msg, err)
					continue
				}
				_ = websiteRepo.Save(context.Background(), &web)
			case "delete":
				if err := w.DeleteWebsite(request.WebsiteDelete{
					ID: web.ID,
				}); err != nil {
					t.LogFailedWithErr(msg, err)
					continue
				}
			}

			t.LogSuccess(msg)
		}
		return nil
	}
	opTask.AddSubTask("", opWebsiteTask, nil)

	go func() {
		_ = opTask.Execute()
	}()
	return nil
}

func (w WebsiteService) BatchSetGroup(req request.BatchWebsiteGroup) error {
	websites, _ := websiteRepo.List(repo.WithByIDs(req.IDs))
	for _, web := range websites {
		web.WebsiteGroupID = req.GroupID
		if err := websiteRepo.Save(context.Background(), &web); err != nil {
			return err
		}
	}
	return nil
}

func (w WebsiteService) BatchSetHttps(ctx context.Context, req request.BatchWebsiteHttps) error {
	websites, _ := websiteRepo.List(repo.WithByIDs(req.IDs))
	opTask, err := task.NewTaskWithOps(i18n.GetMsgByKey("SSL"), task.TaskBatch, task.TaskScopeWebsite, req.TaskID, 0)
	if err != nil {
		return err
	}
	websiteHttpsOp := request.WebsiteHTTPSOp{
		Enable:                true,
		WebsiteSSLID:          req.WebsiteSSLID,
		Type:                  req.Type,
		PrivateKey:            req.PrivateKey,
		Certificate:           req.Certificate,
		PrivateKeyPath:        req.PrivateKeyPath,
		CertificatePath:       req.CertificatePath,
		ImportType:            req.ImportType,
		HttpConfig:            req.HttpConfig,
		SSLProtocol:           req.SSLProtocol,
		Algorithm:             req.Algorithm,
		Hsts:                  req.Hsts,
		HstsIncludeSubDomains: req.HstsIncludeSubDomains,
		HttpsPorts:            req.HttpsPorts,
		Http3:                 req.Http3,
	}
	if req.Type == constant.SSLManual {
		websiteSSL, err := getManualWebsiteSSL(websiteHttpsOp)
		if err != nil {
			return err
		}
		if err = websiteSSLRepo.Create(ctx, &websiteSSL); err != nil {
			return err
		}
		websiteHttpsOp.Type = constant.SSLExisted
		websiteHttpsOp.WebsiteSSLID = websiteSSL.ID
	}
	opWebsiteTask := func(t *task.Task) error {
		for _, web := range websites {
			if web.Type == constant.Stream {
				continue
			}
			websiteHttpsOp.WebsiteID = web.ID
			msg := fmt.Sprintf("%s [%s] %s", i18n.GetMsgByKey("Set"), web.PrimaryDomain, i18n.GetMsgByKey("SSL"))
			if _, err := w.OpWebsiteHTTPS(ctx, websiteHttpsOp); err != nil {
				t.LogFailedWithErr(msg, err)
				continue
			}
			t.LogSuccess(msg)
		}
		return nil
	}

	opTask.AddSubTask("", opWebsiteTask, nil)
	go func() {
		_ = opTask.Execute()
	}()
	return nil
}
