// =============================================================================
// 模块: Firewall 防火墙 (agent/app/api/v2/firewall.go)
// 文件: firewall.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"errors"
	"net/http"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/app/service"
	"github.com/1Panel-dev/1Panel/agent/utils/firewall/filter"
	"github.com/gin-gonic/gin"
)

// @Tags Firewall
// @Summary Load firewall base info
// @Accept json
// @Param request body dto.OperationWithName true "request"
// @Success 200 {object} dto.FirewallSubsystemStatus
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadFirewallBaseInfo  拿防火墙子系统状态（firewalld/ufw/iptables）
// ============================================================
func (b *BaseApi) LoadFirewallBaseInfo(c *gin.Context) {
	var request dto.OperationWithName
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}

	data, err := firewallService.LoadBaseInfo(request.Name)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}

	helper.SuccessWithData(c, data)
}

// @Tags Firewall
// @Summary Operate firewall
// @Accept json
// @Param request body dto.FirewallLifecycleOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/operate [post]
// ============================================================
// OperateFirewall  启/停/重启防火墙
// ============================================================
func (b *BaseApi) OperateFirewall(c *gin.Context) {
	var request dto.FirewallLifecycleOperation
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}

	if err := firewallService.OperateFirewall(request); err != nil {
		helper.InternalServer(c, err)
		return
	}

	helper.Success(c)
}

// @Tags Firewall
// @Summary Load forwarding base info
// @Accept json
// @Success 200 {object} dto.FirewallSubsystemStatus
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadForwardingBaseInfo  拿端口转发子系统状态
// ============================================================
func (b *BaseApi) LoadForwardingBaseInfo(c *gin.Context) {
	data, err := forwardingService.LoadBaseInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags Firewall
// @Summary Page forwarding rules
// @Accept json
// @Param request body dto.ForwardRuleSearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// SearchForwardingRules  分页查端口转发规则
// ============================================================
func (b *BaseApi) SearchForwardingRules(c *gin.Context) {
	var request dto.ForwardRuleSearch
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	total, items, err := forwardingService.SearchRules(request)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}

	helper.SuccessWithData(c, dto.PageResult{Items: items, Total: total})
}

// @Tags Firewall
// @Summary Operate forwarding rules
// @Accept json
// @Param request body dto.ForwardRuleOperate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/forward/operate [post]
// ============================================================
// OperateForwardingRules  批量增删改端口转发规则
// ============================================================
func (b *BaseApi) OperateForwardingRules(c *gin.Context) {
	var request dto.ForwardRuleOperate
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}

	if err := forwardingService.OperateRules(request); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Enable forwarding
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/forward/enable [post]
// ============================================================
// EnableForwarding  初始化并启用端口转发子系统
// ============================================================
func (b *BaseApi) EnableForwarding(c *gin.Context) {
	if err := forwardingService.Enable(); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Apply/Unload/Init firewall filter chain
// @Accept json
// @Param request body dto.FilterChainOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/filter/operate [post]
// ============================================================
// OperateFilterChain  应用/卸载/初始化防火墙过滤链
// ============================================================
func (b *BaseApi) OperateFilterChain(c *gin.Context) {
	var request dto.FilterChainOperation
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if err := firewallService.OperateFilterChain(request); err != nil {
		helper.InternalServer(c, err)
		return
	}

	helper.Success(c)
}

// @Tags Firewall
// @Summary List unified firewall v2 rules
// @Accept json
// @Param request body dto.FirewallRuleInventory true "request"
// @Success 200 {object} dto.FirewallRuleInventoryResponse
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// SearchFirewallRules  列出统一防火墙 v2 规则清单
// ============================================================
func (b *BaseApi) SearchFirewallRules(c *gin.Context) {
	var request dto.FirewallRuleInventory
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	inventory, err := firewallService.Inventory(c.Request.Context(), request)
	if err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.SuccessWithData(c, inventory)
}

// @Tags Firewall
// @Summary Load one provider-native firewall object definition
// @Accept json
// @Param request body dto.FirewallNativeDetail true "request"
// @Success 200 {string} string
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadFirewallNativeDetail  拿"厂商原生"防火墙对象定义（iptables 规则等）
// ============================================================
func (b *BaseApi) LoadFirewallNativeDetail(c *gin.Context) {
	var request dto.FirewallNativeDetail
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	info, err := firewallService.LoadFirewallNativeDetail(c.Request.Context(), request)
	if err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// @Tags Firewall
// @Summary Check unified firewall v2 rules for duplicates and conflicts
// @Accept json
// @Param request body dto.FirewallRuleCheck true "request"
// @Success 200 {object} dto.FirewallRuleCheckResponse
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// CheckFirewallRules  校验防火墙规则冲突/重复（保存前 dry-run）
// ============================================================
func (b *BaseApi) CheckFirewallRules(c *gin.Context) {
	var request dto.FirewallRuleCheck
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	result, err := firewallService.Check(c.Request.Context(), c.ClientIP(), request)
	if err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.SuccessWithData(c, result)
}

// @Tags Firewall
// @Summary Create unified firewall v2 rules
// @Accept json
// @Param request body dto.FirewallRuleCreate true "request"
// @Success 200 {object} dto.FirewallRuleCreateResponse
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/rules [post]
// ============================================================
// CreateFirewallRules  批量创建防火墙 v2 规则
// ============================================================
func (b *BaseApi) CreateFirewallRules(c *gin.Context) {
	var request dto.FirewallRuleCreate
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	result, err := firewallService.Create(c.Request.Context(), request)
	if err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.SuccessWithData(c, result)
}

// @Tags Firewall
// @Summary Delete managed unified firewall v2 rules
// @Accept json
// @Param request body dto.FirewallRuleDelete true "request"
// @Success 200 {object} dto.FirewallRuleDeleteResponse
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/rules/delete [post]
// ============================================================
// DeleteFirewallRules  批量删除防火墙 v2 规则
// ============================================================
func (b *BaseApi) DeleteFirewallRules(c *gin.Context) {
	var request dto.FirewallRuleDelete
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	result, err := firewallService.Delete(c.Request.Context(), request)
	if err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.SuccessWithData(c, result)
}

// @Tags Firewall
// @Summary Update a managed unified firewall v2 rule
// @Accept json
// @Param request body dto.FirewallRuleUpdate true "request"
// @Success 200
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/rules/update [post]
// ============================================================
// UpdateFirewallRule  更新单条防火墙规则
// ============================================================
func (b *BaseApi) UpdateFirewallRule(c *gin.Context) {
	var request dto.FirewallRuleUpdate
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if !normalizeFirewallRuleUUID(c, &request.UUID) {
		return
	}
	if err := firewallService.Update(c.Request.Context(), c.ClientIP(), request); err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Reorder a managed unified firewall v2 rule
// @Accept json
// @Param request body dto.FirewallRuleReorder true "request"
// @Success 200
// @Failure 400 {object} dto.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/rules/reorder [post]
// ============================================================
// ReorderFirewallRule  调整防火墙规则顺序（影响命中优先级）
// ============================================================
func (b *BaseApi) ReorderFirewallRule(c *gin.Context) {
	var request dto.FirewallRuleReorder
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if !normalizeFirewallRuleUUID(c, &request.UUID) {
		return
	}
	if err := firewallService.Reorder(c.Request.Context(), c.ClientIP(), request); err != nil {
		handleFirewallRuleError(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// normalizeFirewallRuleUUID  工具：trim + 校验 UUID 非空
// ============================================================
func normalizeFirewallRuleUUID(c *gin.Context, value *string) bool {
	if value == nil {
		helper.BadRequest(c, repo.ErrFirewallPersistenceInvalid)
		return false
	}
	*value = strings.TrimSpace(*value)
	if *value == "" {
		helper.BadRequest(c, repo.ErrFirewallPersistenceInvalid)
		return false
	}
	return true
}

// ============================================================
// handleFirewallRuleError  把服务层错误映射到 HTTP 状态码 + 业务码
// ============================================================
func handleFirewallRuleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filter.ErrLockoutRisk), errors.Is(err, filter.ErrProtectedRule):
		helper.ErrorWithBusinessCode(c, http.StatusBadRequest, "FW_LOCKOUT_RISK", "ErrInvalidParams", err)
	case errors.Is(err, filter.ErrRuleStale):
		helper.ErrorWithBusinessCode(c, http.StatusConflict, "FW_RULE_STALE", "ErrInvalidParams", err)
	case errors.Is(err, repo.ErrFirewallRuleRevisionConflict):
		helper.ErrorWithBusinessCode(c, http.StatusConflict, "FW_RULE_REVISION_CONFLICT", "ErrInvalidParams", err)
	case errors.Is(err, filter.ErrRuleCheckRequired):
		helper.ErrorWithBusinessCode(c, http.StatusConflict, "FW_RULE_CHECK_REQUIRED", "ErrInvalidParams", err)
	case errors.Is(err, filter.ErrUnsupportedScope), errors.Is(err, filter.ErrInvalidScope),
		errors.Is(err, filter.ErrProviderUnavailable), errors.Is(err, filter.ErrAdapterUnavailable):
		helper.ErrorWithBusinessCode(c, http.StatusBadRequest, "FW_SCOPE_UNSUPPORTED", "ErrInvalidParams", err)
	case errors.Is(err, filter.ErrInvalidRule), errors.Is(err, filter.ErrRuleOperation),
		errors.Is(err, repo.ErrFirewallPersistenceInvalid):
		helper.ErrorWithBusinessCode(c, http.StatusBadRequest, "FW_RULE_UNSUPPORTED", "ErrInvalidParams", err)
	case errors.Is(err, filter.ErrVerificationFailed):
		helper.ErrorWithBusinessCode(c, http.StatusInternalServerError, "FW_VERIFY_FAILED", "ErrInternalServer", err)
	default:
		helper.ErrorWithBusinessCode(c, http.StatusInternalServerError, "FW_APPLY_FAILED", "ErrInternalServer", err)
	}
}

// @Tags Firewall
// @Summary Load firewall settings
// @Success 200 {object} dto.FirewallSettings
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadFirewallSettings  拿防火墙全局设置（启停哪个后端等）
// ============================================================
func (b *BaseApi) LoadFirewallSettings(c *gin.Context) {
	data, err := firewallSettingService.Load(c.Request.Context())
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags Firewall
// @Summary Operate firewall backend
// @Accept json
// @Param request body dto.FirewallBackendOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/settings/operate [post]
// ============================================================
// OperateFirewallBackend  操作防火墙后端（firewalld/ufw/iptables 切换）
// ============================================================
func (b *BaseApi) OperateFirewallBackend(c *gin.Context) {
	var request dto.FirewallBackendOperation
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if err := firewallSettingService.Operate(c.Request.Context(), request); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary List Docker port guard status and policies
// @Success 200 {object} dto.DockerPortGuardList
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// ListDockerPortGuard  列出 Docker 端口防护状态
// ============================================================
func (b *BaseApi) ListDockerPortGuard(c *gin.Context) {
	data, err := dockerPortGuardService.LoadOverview(c.Request.Context())
	if err != nil {
		handleDockerPortGuardError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags Firewall
// @Summary Sync Docker port guard rules
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/docker/sync [post]
// ============================================================
// SyncDockerPortGuard  同步 Docker 端口防护规则（从 docker inspect 重建）
// ============================================================
func (b *BaseApi) SyncDockerPortGuard(c *gin.Context) {
	if err := dockerPortGuardService.Reconcile(c.Request.Context()); err != nil {
		handleDockerPortGuardError(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Operate Docker port guard
// @Accept json
// @Param request body dto.DockerPortGuardOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/docker/operate [post]
// ============================================================
// OperateDockerPortGuard  操作 Docker 端口防护（启停/单条策略）
// ============================================================
func (b *BaseApi) OperateDockerPortGuard(c *gin.Context) {
	var request dto.DockerPortGuardOperation
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if err := dockerPortGuardService.Operate(c.Request.Context(), request); err != nil {
		handleDockerPortGuardError(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Delete Docker port guard policies
// @Accept json
// @Param request body dto.DockerPortGuardPolicyBatchDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/docker/policies/delete/batch [post]
// ============================================================
// DeleteDockerPortGuardPolicies  批量删除 Docker 端口防护策略
// ============================================================
func (b *BaseApi) DeleteDockerPortGuardPolicies(c *gin.Context) {
	var request dto.DockerPortGuardPolicyBatchDelete
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if err := dockerPortGuardService.DeletePolicies(c.Request.Context(), request); err != nil {
		handleDockerPortGuardError(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Firewall
// @Summary Batch upsert Docker port guard policies
// @Accept json
// @Param request body dto.DockerPortGuardPolicyBatch true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/firewall/docker/policies/batch [post]
// ============================================================
// UpsertDockerPortGuardPolicies  批量 upsert Docker 端口防护策略
// ============================================================
func (b *BaseApi) UpsertDockerPortGuardPolicies(c *gin.Context) {
	var request dto.DockerPortGuardPolicyBatch
	if err := helper.CheckBindAndValidate(&request, c); err != nil {
		return
	}
	if err := dockerPortGuardService.UpsertPolicies(c.Request.Context(), request); err != nil {
		handleDockerPortGuardError(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// handleDockerPortGuardError  Docker 端口防护错误处理
// ============================================================
func handleDockerPortGuardError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrDockerGuardInvalid) {
		helper.ErrorWithBusinessCode(c, http.StatusBadRequest, "FW_DOCKER_GUARD_INVALID", "ErrInvalidParams", err)
		return
	}
	if errors.Is(err, service.ErrDockerUnavailable) {
		helper.ErrorWithBusinessCode(c, http.StatusServiceUnavailable, "FW_DOCKER_UNAVAILABLE", "ErrDockerFailed", err)
		return
	}
	helper.ErrorWithBusinessCode(c, http.StatusInternalServerError, "FW_DOCKER_GUARD_FAILED", "ErrInternalServer", err)
}
