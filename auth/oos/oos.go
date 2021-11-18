package oos

import (
	"errors"
	"strings"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gorm_logger_wrapper"
	"github.com/gopherd/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/config"
)

type Service interface {
	Config() *config.Config
}

func New(service Service) interface {
	module.Module
	auth.OOSModule
} {
	return newOOSModule(service)
}

// oosModule implements auth.OOSModule
type oosModule struct {
	*module.BaseModule
	service Service
	db      *gorm.DB
}

func newOOSModule(service Service) *oosModule {
	return &oosModule{
		BaseModule: module.NewBaseModule("oos"),
		service:    service,
	}
}

func (mod *oosModule) Init() error {
	if err := mod.BaseModule.Init(); err != nil {
		return err
	}
	if db, err := gorm.Open(mysql.Open(mod.service.Config().DB.DSN), &gorm.Config{
		Logger: gorm_logger_wrapper.New(log.DefaultLogger, gorm_logger_wrapper.DefaultCalldepth+2),
	}); err != nil {
		return erron.Throw(err)
	} else {
		mod.db = db
	}
	return nil
}

func (mod *oosModule) CreateSchema(obj auth.Object) error {
	return mod.db.AutoMigrate(obj)
}

func formatConds(by []auth.Field) []interface{} {
	if len(by) == 0 {
		return nil
	}
	var sb strings.Builder
	var args = make([]interface{}, 0, len(by)+1)
	args = append(args, nil)
	for i := range by {
		if i > 0 {
			sb.WriteString(" and ")
		}
		sb.WriteByte('`')
		sb.WriteString(by[i].Name)
		sb.WriteString("` = ?")
		args = append(args, by[i].Value)
	}
	args[0] = sb.String()
	return args
}

func (mod *oosModule) GetObject(obj auth.Object, by ...auth.Field) (bool, error) {
	if err := mod.db.Take(obj, formatConds(by)...).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (mod *oosModule) HasObject(tableName string, by ...auth.Field) (bool, error) {
	var count int64
	var err error
	var conds = formatConds(by)
	if len(conds) > 0 {
		err = mod.db.Table(tableName).Where(conds[0], conds[1:]).Count(&count).Error
	} else {
		err = mod.db.Table(tableName).Count(&count).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}

func (mod *oosModule) InsertObject(obj auth.Object) error {
	return mod.db.Create(obj).Error
}

func (mod *oosModule) UpdateObject(obj auth.Object, fields ...interface{}) (int64, error) {
	var result *gorm.DB
	if len(fields) > 0 {
		result = mod.db.Model(obj).Select(fields[0], fields[1:]...).Updates(obj)
	} else {
		result = mod.db.Model(obj).Updates(obj)
	}
	return result.RowsAffected, result.Error
}
