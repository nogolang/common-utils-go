package dtmUtils

import (
	"context"
	"database/sql"

	"github.com/dtm-labs/dtm/client/dtmcli"
	"github.com/dtm-labs/dtm/client/dtmgrpc"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func GetRawSqlTx(begin *gorm.DB) *sql.Tx {
	return begin.Statement.ConnPool.(*sql.Tx)
}

func MustBarrierFromGin(c *gin.Context) (*dtmcli.BranchBarrier, error) {
	barrier, err := dtmcli.BarrierFromQuery(c.Request.URL.Query())
	if err != nil {
		return nil, errors.WithMessage(err, "MustBarrierFromGin出错")
	}
	return barrier, nil
}

func MustBarrierFromGrpc(c context.Context) (*dtmcli.BranchBarrier, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(c)
	if err != nil {
		return nil, errors.WithMessage(err, "MustBarrierFromGrpc出错")
	}
	return barrier, nil
}

// 注意,不能用once,而是每次xa都要创建一个gormDb对象
func RawDbToGormDb(existGormDb *gorm.DB, db *sql.DB) (*gorm.DB, error) {
	//必须要获取当前gormDb的配置才行，不然还要自己去配
	var newGormConfig gorm.Config
	err := copier.Copy(&newGormConfig, existGormDb.Config)
	if err != nil {
		return nil, errors.WithMessage(err, "复制gorm配置出错")
	}
	//transDb, err = gorm.Open(mysql.New(mysql.Config{Conn: db}), &newGormConfig)
	gormDb, err := gorm.Open(mysql.New(mysql.Config{Conn: db}), &newGormConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "dtm转换到gorm数据库连接错误")
	}
	return gormDb, nil
}
