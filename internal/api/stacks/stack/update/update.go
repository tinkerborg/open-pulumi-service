package update

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
	"github.com/tinkerborg/open-pulumi-service/pkg/router/middleware"
)

// TODO - currently the stack name doesn't matter as long as updateID is correct,
//
//	but the updateID should have to correspond to the stack in the path
func Setup(p *state.Service, prefix *middleware.DynamicPrefix[client.StackIdentifier]) router.Setup {
	return func(r *router.Router) {
		updateIdentifier := middleware.NewDynamicPrefix("/{updateKind}/{updateID}",
			func(r *http.Request) (client.UpdateIdentifier, error) {
				// TODO check path params
				updateKind := r.PathValue("updateKind")
				updateID := r.PathValue("updateID")

				return client.UpdateIdentifier{
					StackIdentifier: prefix.Value(r),
					UpdateKind:      apitype.UpdateKind(updateKind),
					UpdateID:        updateID,
				}, nil
			})

		r.Use(updateIdentifier.Middleware)

		r.GET("/{$}", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			results, err := p.GetUpdateResults(identifier)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return nil, &router.HTTPError{Code: 404, Message: "update not found"}
				}
				return nil, err
			}

			return apitype.UpdateResults{
				Status: results.Status,
				Events: []apitype.UpdateEvent{},
			}, nil
		})

		r.POST("/", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			var request apitype.StartUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid stack"}
			}

			version, err := p.StartUpdate(identifier)
			if err != nil {
				return nil, err
			}
			// tokenExpiration: Math.floor(new Date().getTime() / 1000) + 86400
			return apitype.StartUpdateResponse{
				Version: version,
				// used in update complete/event requests w/ auth header "update-token X",
				// should be a JWT:
				// Token header
				// ------------
				// {
				//   "typ": "JWT",
				//   "alg": "HS256"
				// }

				// Token claims
				// ------------
				// {
				//   "exp": 1758981121,
				//   "updateID": "d23b9399-120e-4034-ade0-07eee23aa9e6"
				// }
				Token: "foo",
			}, nil
		})

		r.PATCH("/checkpoint", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			var request apitype.PatchUpdateCheckpointRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid checkpoint request"}
			}

			checkpoint := &apitype.VersionedCheckpoint{
				Version:    request.Version,
				Features:   request.Features,
				Checkpoint: request.Deployment,
			}

			if err := p.CheckpointUpdate(identifier, checkpoint); err != nil {
				return nil, err
			}

			// TODO - figure out what this response should actually be
			return model.CompleteUpdateResponse{
				Version: 2,
			}, nil
		})

		// TODO - what happens on official API when you start an update, start and complete a different update, and then
		//        complete the first udpate? how do version numbers work?
		r.POST("/complete", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			var request *apitype.CompleteUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid stack"}
			}

			version, err := p.CompleteUpdate(identifier, request.Status)
			if err != nil {
				return nil, err
			}

			return model.CompleteUpdateResponse{
				Version: *version,
			}, nil
		})

		// TODO filtering
		r.GET("/events", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			events, err := p.ListEngineEvents(identifier)
			if err != nil {
				return nil, err
			}

			return events, nil
		})

		r.POST("/events/batch", func(r *http.Request) (any, error) {
			identifier := updateIdentifier.Value(r)

			var request apitype.EngineEventBatch
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "Invalid stack"}
			}

			if err := p.AddEngineEvents(identifier, request.Events); err != nil {
				return nil, err
			}

			return nil, nil
		})

	}
}
