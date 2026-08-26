// =============================================================================
// 模块: Firewall 防火墙 (agent/app/repo/firewall_rule.go)
// 文件: firewall_rule.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrFirewallRuleRevisionConflict = errors.New("firewall rule revision conflict")
	ErrFirewallPersistenceInvalid   = errors.New("invalid firewall persistence record")
)

// ============================================================
// IFirewallRuleRepo  防火墙 v2 规则数据库访问接口
// ============================================================
// 方法: Create / GetByUUID / List / UpdateWithRevision / DeleteWithRevision
// ============================================================
type IFirewallRuleRepo interface {
	Create(context.Context, *model.FirewallRule) error
	GetByUUID(context.Context, string) (model.FirewallRule, error)
	List(context.Context, ...DBOption) ([]model.FirewallRule, error)
	UpdateWithRevision(context.Context, string, uint, map[string]interface{}) error
	DeleteWithRevision(context.Context, string, uint) error
}

// ============================================================
// FirewallRuleRepo  防火墙规则 GORM 仓库
// ============================================================
// 字段:
//   - db (*gorm.DB) — DB 句柄（可能为 nil，自动用全局）
// ============================================================
type FirewallRuleRepo struct {
	db *gorm.DB
}

// ============================================================
// WithFirewallRuleScope  构造"按 scope_key 过滤"的 DBOption
// ============================================================
func WithFirewallRuleScope(scopeKey string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("scope_key = ?", scopeKey)
	}
}

// ============================================================
// WithFirewallRuleSource  构造"按 owner 过滤"的 DBOption
// ============================================================
func WithFirewallRuleSource(kind, id string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("owner = ?", model.FirewallRuleOwner(kind, id))
	}
}

// ============================================================
// NewIFirewallRuleRepo  构造接口实现
// ============================================================
func NewIFirewallRuleRepo() IFirewallRuleRepo {
	return &FirewallRuleRepo{}
}

// ============================================================
// NewFirewallRuleRepo  构造具体实现（注入 DB）
// ============================================================
func NewFirewallRuleRepo(db *gorm.DB) *FirewallRuleRepo {
	return &FirewallRuleRepo{db: db}
}

// ============================================================
// Create  新增一条防火墙规则（带必填校验、UUID/Revision 默认填充）
// ============================================================
func (r *FirewallRuleRepo) Create(ctx context.Context, rule *model.FirewallRule) error {
	if err := prepareFirewallRule(rule); err != nil {
		return err
	}
	return r.dbFor(ctx).Create(rule).Error
}

// ============================================================
// GetByUUID  按 UUID 查一条规则
// ============================================================
func (r *FirewallRuleRepo) GetByUUID(ctx context.Context, ruleUUID string) (model.FirewallRule, error) {
	var rule model.FirewallRule
	err := r.dbFor(ctx).Where("uuid = ?", ruleUUID).First(&rule).Error
	return rule, err
}

// ============================================================
// List  按条件列规则
// ============================================================
func (r *FirewallRuleRepo) List(ctx context.Context, opts ...DBOption) ([]model.FirewallRule, error) {
	var rules []model.FirewallRule
	db := r.dbFor(ctx).Model(&model.FirewallRule{})
	for _, opt := range opts {
		db = opt(db)
	}
	return rules, db.Find(&rules).Error
}

// ============================================================
// UpdateWithRevision  按"乐观锁"更新：expectedRevision 不一致就 409
// ============================================================
func (r *FirewallRuleRepo) UpdateWithRevision(ctx context.Context, ruleUUID string, expectedRevision uint, updates map[string]interface{}) error {
	updates = sanitizeRuleUpdates(updates)
	updates["revision"] = gorm.Expr("revision + 1")
	result := r.dbFor(ctx).Model(&model.FirewallRule{}).
		Where("uuid = ? AND revision = ?", ruleUUID, expectedRevision).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFirewallRuleRevisionConflict
	}
	return nil
}

// ============================================================
// DeleteWithRevision  按乐观锁删除
// ============================================================
func (r *FirewallRuleRepo) DeleteWithRevision(ctx context.Context, ruleUUID string, expectedRevision uint) error {
	result := r.dbFor(ctx).
		Where("uuid = ? AND revision = ?", ruleUUID, expectedRevision).
		Delete(&model.FirewallRule{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFirewallRuleRevisionConflict
	}
	return nil
}

func (r *FirewallRuleRepo) dbFor(ctx context.Context) *gorm.DB {
	return firewallDB(ctx, r.db)
}

func firewallDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx, ok := ctx.Value(constant.DB).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	if fallback == nil {
		fallback = global.DB
	}
	return fallback.WithContext(ctx)
}

// ============================================================
// prepareFirewallRule  创建前的字段校验 + 默认值填充
// ============================================================
func prepareFirewallRule(rule *model.FirewallRule) error {
	if rule == nil {
		return fmt.Errorf("%w: rule is nil", ErrFirewallPersistenceInvalid)
	}
	if rule.ScopeKey == "" || rule.Provider == "" || rule.Family == "" || rule.Location == "" ||
		rule.NativeKind == "" || rule.Protocol == "" || rule.Action == "" || rule.RuleKey == "" {
		return fmt.Errorf("%w: atomic rule identity fields are required", ErrFirewallPersistenceInvalid)
	}
	if rule.UUID == "" {
		rule.UUID = uuid.NewString()
	}
	if rule.Revision == 0 {
		rule.Revision = 1
	}
	if rule.Origin == "" {
		rule.Origin = constant.FirewallRuleOriginCreated
	}
	if rule.Owner == "" {
		rule.Owner = constant.FirewallRuleSourceUser
	}
	return nil
}

// ============================================================
// sanitizeRuleUpdates  过滤不允许客户端直接改的字段（id/uuid/revision/created_at）
// ============================================================
func sanitizeRuleUpdates(updates map[string]interface{}) map[string]interface{} {
	result := cloneUpdates(updates)
	delete(result, "id")
	delete(result, "uuid")
	delete(result, "revision")
	delete(result, "created_at")
	return result
}

func cloneUpdates(updates map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(updates)+1)
	for key, value := range updates {
		result[key] = value
	}
	return result
}
