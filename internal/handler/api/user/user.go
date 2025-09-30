package user

import (
	"net/http"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(p *state.Service) router.Setup {
	return func(r *router.Router) {
		r.GET("/", func(w *router.ResponseWriter, r *http.Request) error {
			return w.JSON(p.GetCurrentUser())
		})

		r.GET("/stacks/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			stacks, err := p.ListUserStacks()
			if err != nil {
				return w.Error(err)
			}

			return w.JSON(&apitype.ListStacksResponse{Stacks: stacks})
		})
	}
}
