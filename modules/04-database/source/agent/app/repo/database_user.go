// =============================================================================
// 模块: Database 数据库 (agent/app/repo/database_user.go)
// 文件: database_user.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"context"
	"fmt"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	"gorm.io/gorm"
)

// DatabaseUserRepo (struct)
type DatabaseUserRepo struct{}

// IDatabaseUserRepo (interface)
type IDatabaseUserRepo interface {
	Get(opts ...DBOption) (model.DatabaseUser, error)
	List(opts ...DBOption) ([]model.DatabaseUser, error)
	Save(user *model.DatabaseUser) error
	Delete(opts ...DBOption) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
	Update(vars map[string]interface{}, opts ...DBOption) error
	WithByDatabase(database string) DBOption
	WithByUser(username, host string) DBOption
	WithByUserList(users [][2]string) DBOption
}

func NewIDatabaseUserRepo() IDatabaseUserRepo {
	return &DatabaseUserRepo{}
}

func (u *DatabaseUserRepo) Get(opts ...DBOption) (model.DatabaseUser, error) {
	var user model.DatabaseUser
	db := global.DB.Model(&model.DatabaseUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&user).Error; err != nil {
		return user, err
	}
	password, err := encrypt.StringDecrypt(user.Password)
	if err != nil {
		global.LOG.Errorf("decrypt database user %s password failed, err: %v", user.Username, err)
	}
	user.Password = password
	return user, nil
}

func (u *DatabaseUserRepo) List(opts ...DBOption) ([]model.DatabaseUser, error) {
	var users []model.DatabaseUser
	db := global.DB.Model(&model.DatabaseUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Find(&users).Error; err != nil {
		return users, err
	}
	for i := 0; i < len(users); i++ {
		password, err := encrypt.StringDecrypt(users[i].Password)
		if err != nil {
			global.LOG.Errorf("decrypt database user %s password failed, err: %v", users[i].Username, err)
		}
		users[i].Password = password
	}
	return users, nil
}

func (u *DatabaseUserRepo) Save(user *model.DatabaseUser) error {
	if len(user.Password) != 0 {
		password, err := encrypt.StringEncrypt(user.Password)
		if err != nil {
			return fmt.Errorf("encrypt database user %s password failed, err: %v", user.Username, err)
		}
		user.Password = password
	}
	return global.DB.Save(user).Error
}

func (u *DatabaseUserRepo) Delete(opts ...DBOption) error {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.DatabaseUser{}).Error
}

func (u *DatabaseUserRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.DatabaseUser{}).Error
}

func (u *DatabaseUserRepo) Update(vars map[string]interface{}, opts ...DBOption) error {
	db := global.DB.Model(&model.DatabaseUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Updates(vars).Error
}

func (u *DatabaseUserRepo) WithByDatabase(database string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("database = ?", database)
	}
}

func (u *DatabaseUserRepo) WithByUser(username, host string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("username = ? AND host = ?", username, host)
	}
}

func (u *DatabaseUserRepo) WithByUserList(users [][2]string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(users) == 0 {
			return g.Where("1 = 0")
		}
		values := make([][]interface{}, 0, len(users))
		for _, user := range users {
			values = append(values, []interface{}{user[0], user[1]})
		}
		return g.Where("(username, host) IN ?", values)
	}
}
