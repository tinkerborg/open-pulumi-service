package user

import (
	"net/http"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(p *state.Service) router.Setup {
	return func(r *router.Router) {
		r.GET("/", func(r *http.Request) (interface{}, error) {
			return p.GetCurrentUser()
		})

		r.GET("/stacks/{$}", func(r *http.Request) (interface{}, error) {
			stacks, err := p.ListUserStacks()
			if err != nil {
				return nil, err
			}

			return &apitype.ListStacksResponse{Stacks: stacks}, nil
		})
	}
}
