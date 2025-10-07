package model

import "github.com/pulumi/pulumi/sdk/v3/go/common/apitype"

type ListStackResourcesResponse struct {
	Resources []apitype.ResourceV3 `json:"resources"`
	Region    string               `json:"region"`
	Version   int                  `json:"version"`
}
