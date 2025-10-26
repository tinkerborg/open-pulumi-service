package store

import (
	"gorm.io/gorm"
)

// TODO this is a stupid place for this
func parseDefaultOptions(db *gorm.DB, model interface{}) []DBOption {
	options := []DBOption{}

	stmt := &gorm.Statement{DB: db}

	stmt.Parse(model)

	for _, field := range stmt.Schema.Fields {
		// TODO desc
		_, ok := field.TagSettings["ORDERBY"]
		if ok {
			options = append(options, OrderBy(field.DBName))
		}
	}

	return options
}
