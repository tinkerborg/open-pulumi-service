package store

import (
	"context"
	"reflect"

	"github.com/tinkerborg/open-pulumi-service/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DBOption interface {
	apply(db *gorm.DB, record interface{}) (*gorm.DB, error)
}

// Where option
type where struct {
	query interface{}
}

func Where(query interface{}) *where {
	return &where{query}
}

func (w *where) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	return db.Where(w.query), nil
}

// OrderBy option
type orderByKey string

const (
	orderByColumn orderByKey = "order_column"
)

type orderBy struct {
	order string
}

func OrderBy(order string) *orderBy {
	return &orderBy{order}
}

func (w *orderBy) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	ctx := context.WithValue(db.Statement.Context, orderByColumn, w.order)

	if !util.IsSlice(record) {
		return db, nil
	}

	return db.WithContext(ctx).Order(w.order), nil
}

// Descending option
type descending struct {
	enabled bool
}

func Descending(enabled ...bool) *descending {
	return &descending{
		enabled: len(enabled) == 0 || enabled[len(enabled)-1],
	}
}

func (d *descending) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	if !d.enabled {
		return db, nil
	}

	column, exists := db.Statement.Context.Value(orderByColumn).(string)
	if !exists {
		return db, nil
	}

	return db.Clauses(clause.OrderBy{
		Columns: []clause.OrderByColumn{
			{
				Column:  clause.Column{Name: column},
				Desc:    true,
				Reorder: true,
			},
		},
	}), nil
}

// Limit option
type limit struct {
	count int
}

func Limit(count int) *limit {
	return &limit{count}
}

func (l *limit) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	if l.count == 0 {
		return db, nil
	}

	return db.Limit(l.count), nil
}

// Offset option
type offset struct {
	count int
}

func Offset(count int) *offset {
	return &offset{count}
}

func (l *offset) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	if l.count == 0 {
		return db, nil
	}

	return db.Offset(l.count), nil
}

// join option
type join struct {
	column  interface{}
	options []DBOption
}

func Join(column interface{}, options ...DBOption) *join {
	return &join{column, options}
}

func (j *join) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	preloadDb := db
	preloadDb, err := applyOptions(preloadDb, j.column, j.options...)
	if err != nil {
		return nil, err
	}

	preloadOptions := func(d *gorm.DB) *gorm.DB {
		return preloadDb
	}

	if _, ok := j.column.(string); ok {
		return db.Joins(j.column.(string), preloadOptions), nil
	}

	field, err := getFieldOfType(record, j.column)
	if err != nil {
		return nil, err
	}

	return db.Joins(field, j.column, preloadOptions), nil
}

// End options

func applyOptions(db *gorm.DB, record interface{}, opts ...DBOption) (*gorm.DB, error) {
	key := getValueType(record).Type()
	if key.Kind() == reflect.Slice {
		key = key.Elem()
	}

	if defaults, ok := defaultOptions[key]; ok {
		opts = append(defaults, opts...)
	}

	for _, opt := range opts {
		dbWithOptions, err := opt.apply(db, record)
		if err != nil {
			return nil, err
		}
		db = dbWithOptions
	}
	return db, nil
}

type debug struct{}

func (d *debug) apply(db *gorm.DB, record interface{}) (*gorm.DB, error) {
	return db.Debug(), nil
}

var Debug = &debug{}

func WithLimit(limit int) func(*gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		if limit == 0 {
			return d
		}
		return d.Limit(limit)
	}
}

func WithOrder(order string) func(*gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		if order == "" {
			return d
		}
		return d.Order(order)
	}
}
