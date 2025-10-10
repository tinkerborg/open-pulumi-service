package user

import (
	"net/http"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/service/auth"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(a *auth.Service, p *state.Service) router.Setup {
	return func(r *router.Router) {
		r.GET("/", func(w *router.ResponseWriter, r *http.Request) error {
			claims, err := a.GetRequestClaims(r)
			if err != nil {
				return w.Error(err)
			}

			user, err := p.GetUser(claims.ID)
			if err != nil {
				return w.WithStatus(http.StatusInternalServerError).Error(err)
			}

			return w.JSON(user)
		})

		r.GET("/organizations/default/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			w.JSON(&apitype.GetDefaultOrganizationResponse{
				GitHubLogin: "cimulate-ai",
				Messages: []apitype.Message{
					{Message: "Hello world"},
				},
			})
			return nil
		})

		r.GET("/stacks/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			// TODO tag
			query := r.URL.Query()
			organization := query.Get("organization")
			project := query.Get("project")

			// TODO - should use default org
			stacks, err := p.ListUserStacks(model.StackRecord{Owner: organization, Project: project})
			if err != nil {
				return w.Error(err)
			}

			return w.JSON(&apitype.ListStacksResponse{Stacks: stacks})
		})
	}
}
