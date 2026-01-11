package dataUtils

import (
	"database/sql"
	"time"

	"github.com/jinzhu/copier"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var CopierTypeConverters = []copier.TypeConverter{
	timeToGrpcTimestamp,
	timePointerToGrpcTimestamp,
	sqlNullTimeToGrpcTimestamp,
	gormDeletedAtToGrpcTimestamp,
	grpcTimestampToTime,
	grpcTimestampToSqlNullTime,
	grpcTimestampToTimePointer,
}

var timeToGrpcTimestamp = copier.TypeConverter{
	SrcType: time.Time{},
	DstType: &timestamppb.Timestamp{},
	Fn: func(src interface{}) (interface{}, error) {
		timeRaw, ok := src.(time.Time)
		if !ok {
			return nil, nil
		}
		if !timeRaw.IsZero() {
			return timestamppb.New(timeRaw), nil
		}
		return nil, nil
	},
}
var timePointerToGrpcTimestamp = copier.TypeConverter{
	SrcType: &time.Time{},
	DstType: &timestamppb.Timestamp{},
	Fn: func(src interface{}) (interface{}, error) {
		timeRaw, ok := src.(*time.Time)
		if !ok {
			return nil, nil
		}
		if timeRaw != nil {
			return timestamppb.New(*timeRaw), nil
		}
		return nil, nil
	},
}

var sqlNullTimeToGrpcTimestamp = copier.TypeConverter{
	SrcType: sql.NullTime{},
	DstType: &timestamppb.Timestamp{},
	Fn: func(src interface{}) (interface{}, error) {
		timeRaw, ok := src.(sql.NullTime)
		if !ok {
			return nil, nil
		}
		if timeRaw.Valid && !timeRaw.Time.IsZero() {
			return timestamppb.New(timeRaw.Time), nil
		}
		return nil, nil
	},
}

var gormDeletedAtToGrpcTimestamp = copier.TypeConverter{
	SrcType: gorm.DeletedAt{},
	DstType: &timestamppb.Timestamp{},
	Fn: func(src interface{}) (interface{}, error) {
		timeRaw, ok := src.(gorm.DeletedAt)
		if !ok {
			return nil, nil
		}
		if timeRaw.Valid && !timeRaw.Time.IsZero() {
			return timestamppb.New(timeRaw.Time), nil
		}
		return nil, nil
	},
}

var grpcTimestampToTime = copier.TypeConverter{
	SrcType: &timestamppb.Timestamp{},
	DstType: time.Time{},
	Fn: func(src interface{}) (interface{}, error) {
		grpcTime, ok := src.(*timestamppb.Timestamp)
		if !ok {
			return nil, nil
		}
		if grpcTime != nil {
			return grpcTime.AsTime(), nil
		}
		return nil, nil
	},
}
var grpcTimestampToSqlNullTime = copier.TypeConverter{
	SrcType: &timestamppb.Timestamp{},
	DstType: sql.NullTime{},
	Fn: func(src interface{}) (interface{}, error) {
		grpcTime, ok := src.(*timestamppb.Timestamp)
		if !ok {
			return nil, nil
		}
		if grpcTime != nil {
			return sql.NullTime{Time: grpcTime.AsTime(), Valid: true}, nil
		}
		return nil, nil
	},
}

var grpcTimestampToTimePointer = copier.TypeConverter{
	SrcType: &timestamppb.Timestamp{},
	DstType: &time.Time{},
	Fn: func(src interface{}) (interface{}, error) {
		grpcTime, ok := src.(*timestamppb.Timestamp)
		if !ok {
			return nil, nil
		}
		if grpcTime != nil {
			asTime := grpcTime.AsTime()
			return &asTime, nil
		}
		return nil, nil
	},
}
