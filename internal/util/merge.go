package util

import (
	"dario.cat/mergo"
)

func Merge[T interface{}](dest T, sources []T) (T, error) {
	for _, source := range sources {
		if err := mergo.Merge(&dest, source, mergo.WithAppendSlice); err != nil {
			return dest, err
		}
	}

	return dest, nil
}
