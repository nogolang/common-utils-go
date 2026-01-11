package dtmUtils

import (
	"context"
	"database/sql"

	"github.com/dtm-labs/dtm/client/dtmcli"
	"github.com/dtm-labs/dtm/client/dtmgrpc"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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
