package model

import (
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/config"
)

// TODO - maybe remove this
func NewBackendConfig(apiConfig map[string]apitype.ConfigValue) (config.Map, error) {
	c := make(config.Map)
	for rawK, rawV := range apiConfig {
		k, err := config.ParseKey(rawK)
		if err != nil {
			return nil, err
		}
		if rawV.Object {
			if rawV.Secret {
				c[k] = config.NewSecureObjectValue(rawV.String)
			} else {
				c[k] = config.NewObjectValue(rawV.String)
			}
		} else {
			if rawV.Secret {
				c[k] = config.NewSecureValue(rawV.String)
			} else {
				c[k] = config.NewValue(rawV.String)
			}
		}
	}
	return c, nil
}
