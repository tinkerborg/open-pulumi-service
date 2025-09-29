package api

import (
	"github.com/tinkerborg/open-pulumi-service/internal/api/stacks"
	"github.com/tinkerborg/open-pulumi-service/internal/api/user"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

func Setup(s *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		r.Mount("/user/", user.Setup(s))
		r.Mount("/stacks/", stacks.Setup(s, c))
	}
}
