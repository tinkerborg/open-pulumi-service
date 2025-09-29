package model

import "github.com/pulumi/pulumi/sdk/v3/go/common/apitype"

type DeploymentResponseV3 struct {
	Version    int                  `json:"version" yaml:"version"`
	Deployment apitype.DeploymentV3 `json:"deployment" yaml:"deployment"`
}
