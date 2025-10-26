package store

import (
	"errors"
	"log"
	"os"
	"reflect"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Postgres struct {
	db          *gorm.DB
	primaryKeys map[interface{}][]string
}

type Model[T any] interface {
	ID()
}

func NewPostgres(connectionString string) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		NamingStrategy: schema.NamingStrategy{

			SingularTable: true,
		},
	})

	if err != nil {
		return nil, err
	}

	return &Postgres{
		db:          db,
		primaryKeys: map[interface{}][]string{},
	}, nil
}

func (p *Postgres) Create(record interface{}) error {
	err := p.db.Create(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrExist
	}

	return err
}

func (p *Postgres) Read(record interface{}, opts ...DBOption) error {
	if err := p.validatePrimaryKey(record); err != nil {
		return err
	}

	db, err := applyOptions(p.db, record, opts...)
	if err != nil {
		return err
	}

	if err := db.First(ensurePtr(record), record).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return nil
}

func (p *Postgres) List(records interface{}, opts ...DBOption) error {
	db, err := applyOptions(p.db, records, opts...)
	if err != nil {
		return err
	}

	err = db.Find(ensurePtr(records)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (p *Postgres) Update(record interface{}) error {
	err := p.db.Save(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (p *Postgres) Delete(record interface{}) error {
	err := p.db.Delete(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (p *Postgres) Count(records interface{}, opts ...DBOption) (int64, error) {
	db, err := applyOptions(p.db, records, opts...)
	if err != nil {
		return -1, err
	}

	var count int64

	err = db.Model(ensurePtr(records)).Where(records).Count(&count).Error

	return count, err
}

func (p *Postgres) Transaction(fc func(p *Postgres) error) error {
	err := p.db.Transaction(func(tx *gorm.DB) error {
		db := &Postgres{db: tx, primaryKeys: p.primaryKeys}
		return fc(db)
	})
	return err
}

func ensurePtr(value interface{}) interface{} {
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		return value
	}
	return &value
}
