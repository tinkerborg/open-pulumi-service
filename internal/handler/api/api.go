package api

import (
	"github.com/tinkerborg/open-pulumi-service/internal/handler/api/stacks"
	"github.com/tinkerborg/open-pulumi-service/internal/handler/api/user"
	"github.com/tinkerborg/open-pulumi-service/internal/service/auth"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(a *auth.Service, s *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		r.Use(a.Middleware)
		r.Mount("/user/", user.Setup(a, s))
		r.Mount("/stacks/", stacks.Setup(a, s, c))
	}
}
