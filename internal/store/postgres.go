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
	})

	if err != nil {
		return nil, err
	}

	primaryKeys := map[interface{}][]string{}

	return &Postgres{db, primaryKeys}, nil
}

func (p *Postgres) Create(record interface{}) error {
	err := p.db.Create(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrExist
	}

	return err
}

func (p *Postgres) Read(record interface{}) error {
	if err := p.validatePrimaryKey(record); err != nil {
		return err
	}

	err := p.db.First(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (p *Postgres) List(records interface{}, conditions ...interface{}) error {
	err := p.db.Find(ensurePtr(records), conditions...).Error
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

func (p *Postgres) Transaction(fc func(p *Postgres) error) error {
	p.db.Transaction(func(tx *gorm.DB) error {
		db := &Postgres{db: tx}
		return fc(db)
	})
	return nil
}

func ensurePtr(value interface{}) interface{} {
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		return value
	}
	return &value
}
