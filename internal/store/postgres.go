package store

import (
	"errors"
	"log"
	"reflect"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Postgres struct {
	db *gorm.DB
}

type Model[T any] interface {
	ID()
}

func NewPostgres(connectionString string) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		TranslateError: true,
		// Logger:         logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, err
	}

	return &Postgres{db}, nil
}

func (p *Postgres) Create(record interface{}) error {
	err := p.db.Create(ensurePtr(record)).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrExist
	}

	return err
}

func (p *Postgres) Read(record interface{}) error {
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

func (p *Postgres) RegisterSchemas(schemas ...interface{}) error {
	for _, schema := range schemas {
		if err := p.db.AutoMigrate(ensurePtr(schema)); err != nil {
			log.Fatalf("schema auto-migration failed: %s", err)
		}
	}

	return nil
}

func ensurePtr(value interface{}) interface{} {
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		return value
	}
	return &value
}
