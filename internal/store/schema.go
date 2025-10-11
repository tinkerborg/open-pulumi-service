package store

import (
	"fmt"
	"log"
	"reflect"

	"gorm.io/gorm"
)

func (p *Postgres) RegisterModels(models ...interface{}) error {
	if err := p.db.AutoMigrate(models...); err != nil {
		log.Fatalf("schema auto-migration failed: %s", err)
	}

	for _, model := range models {
		primaryKeys, err := p.getPrimaryKeys(model)
		if err != nil {
			return err
		}

		p.primaryKeys[getValueType(model).Type()] = primaryKeys
	}

	return nil
}

func (p *Postgres) getPrimaryKeys(model interface{}) ([]string, error) {
	stmt := &gorm.Statement{DB: p.db}

	stmt.Parse(model)
	if stmt.Schema == nil {
		return nil, fmt.Errorf("can't find primary key(s) for model '%s'", reflect.TypeOf(model).Name())
	}

	primaryKeys := []string{}

	for _, field := range stmt.Schema.Fields {
		if field.PrimaryKey || field.TagSettings["UNIQUEINDEX"] != "" || field.TagSettings["INDEX"] != "" {
			primaryKeys = append(primaryKeys, field.Name)
		}
	}

	return primaryKeys, nil
}

func (p *Postgres) validatePrimaryKey(model interface{}) error {
	value := getValueType(model)
	keys := p.primaryKeys[value.Type()]

	for _, key := range keys {
		if !value.FieldByName(key).IsZero() {
			return nil
		}
	}

	return fmt.Errorf("missing primary key for model '%s', keys: %+v", value.Type().Name(), keys)
}

func getValueType(v interface{}) reflect.Value {
	t := reflect.ValueOf(v)
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

func getFieldOfType(input interface{}, matcher interface{}) (string, error) {
	if value := reflect.ValueOf(input); value.Kind() == reflect.Ptr {
		input = value.Elem().Interface()
	}

	inputType := reflect.TypeOf(input)
	matchType := reflect.TypeOf(matcher)

	for idx := range inputType.NumField() {
		field := inputType.Field(idx)
		if field.Type.Kind() == reflect.Slice && field.Type.Elem() == matchType {
			return field.Name, nil
		}
	}

	return "", fmt.Errorf("field not found")
}
