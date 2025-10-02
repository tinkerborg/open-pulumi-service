package stacks

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/handler/api/stacks/stack"
	"github.com/tinkerborg/open-pulumi-service/internal/service/auth"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(a *auth.Service, p *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		r.Mount("/", stack.Setup(a, p, c))

		r.POST("/{owner}/{project}/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			owner := r.PathValue("owner")
			project := r.PathValue("project")

			var request *apitype.CreateStackRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid stack: %s", err)
			}

			stack := &apitype.Stack{
				OrgName:     owner,
				ProjectName: project,
				StackName:   tokens.QName(request.StackName),
				Tags:        request.Tags,
				Version:     0,
				Config:      request.Config,
			}

			if err := p.CreateStack(stack); err != nil {
				if errors.Is(err, store.ErrExist) {
					return w.WithStatus(http.StatusConflict).Errorf("stack already exists")
				}
				return w.Error(err)
			}

			return w.JSON(&apitype.CreateStackResponse{})
		})

	}
}
