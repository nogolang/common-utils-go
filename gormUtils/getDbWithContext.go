package gormUtils

import (
	"context"

	"gorm.io/gorm"
)

func GetDbFromCtx(ctx context.Context) (*gorm.DB, error) {
	db := ctx.Value("db").(*gorm.DB)
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	return db, nil
}
func SetDbToCtx(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, "db", db)
}
