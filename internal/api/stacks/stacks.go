package stacks

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/api/stacks/stack"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(p *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		r.Mount("/", stack.Setup(p, c))

		r.POST("/{owner}/{project}/{$}", func(r *http.Request) (any, error) {
			owner := r.PathValue("owner")
			project := r.PathValue("project")

			var request *apitype.CreateStackRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid stack"}
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
					return nil, &router.HTTPError{Code: 409, Message: "stack already exists"}
				}
				return nil, err
			}

			return &apitype.CreateStackResponse{}, nil
		})

	}
}
