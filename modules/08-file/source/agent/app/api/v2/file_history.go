// =============================================================================
// 模块: File 文件管理 (agent/app/api/v2/file_history.go)
// 文件: file_history.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/gin-gonic/gin"
)

// @Tags File
// @Summary Load file history list
// @Accept json
// @Param request body request.FileHistorySearchReq true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/history/search [post]
func (b *BaseApi) SearchFileHistory(c *gin.Context) {
	var req request.FileHistorySearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, items, err := fileHistoryService.Search(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{Items: items, Total: total})
}

// @Tags File
// @Summary Load file history content
// @Accept json
// @Param request body request.FileHistoryContentReq true "request"
// @Success 200 {object} response.FileHistoryInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/history/content [post]
func (b *BaseApi) GetFileHistoryContent(c *gin.Context) {
	var req request.FileHistoryContentReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	item, err := fileHistoryService.GetContent(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, item)
}

// @Tags File
// @Summary Delete file history record
// @Accept json
// @Param request body request.FileHistoryDeleteReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/history/del [post]
func (b *BaseApi) DeleteFileHistory(c *gin.Context) {
	var req request.FileHistoryDeleteReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := fileHistoryService.Delete(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags File
// @Summary Restore file history record
// @Accept json
// @Param request body request.FileHistoryRestoreReq true "request"
// @Success 200 {object} response.FileInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/history/restore [post]
func (b *BaseApi) RestoreFileHistory(c *gin.Context) {
	var req request.FileHistoryRestoreReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	info, err := fileHistoryService.Restore(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}
