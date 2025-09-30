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

		r.GET("/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			results, err := p.GetUpdateResults(identifier)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return w.WithStatus(http.StatusNotFound).Errorf("stack not found")
				}
				return w.Error(err)
			}

			return w.JSON(apitype.UpdateResults{
				Status: results.Status,
				Events: []apitype.UpdateEvent{},
			})
		})

		r.POST("/", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			var request apitype.StartUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid request: %s", err)
			}

			version, err := p.StartUpdate(identifier)
			if err != nil {
				return w.Errorf("failed to start update: %s", err)
			}

			// tokenExpiration: Math.floor(new Date().getTime() / 1000) + 86400
			return w.JSON(apitype.StartUpdateResponse{
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
			})
		})

		r.PATCH("/checkpoint", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			var request apitype.PatchUpdateCheckpointRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid checkpoint: %s", err)
			}

			checkpoint := &apitype.VersionedCheckpoint{
				Version:    request.Version,
				Features:   request.Features,
				Checkpoint: request.Deployment,
			}

			if err := p.CheckpointUpdate(identifier, checkpoint); err != nil {
				return w.Errorf("checkpoint failed: %s", err)
			}

			// TODO - figure out what this response should actually be
			return w.JSON(model.CompleteUpdateResponse{
				Version: 2,
			})
		})

		// TODO - what happens on official API when you start an update, start and complete a different update, and then
		//        complete the first udpate? how do version numbers work?
		r.POST("/complete", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			var request *apitype.CompleteUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid request: %s", err)
			}

			version, err := p.CompleteUpdate(identifier, request.Status)
			if err != nil {
				return w.Errorf("failed to complete update: %s", err)
			}

			return w.JSON(model.CompleteUpdateResponse{
				Version: *version,
			})
		})

		// TODO filtering
		r.GET("/events", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			events, err := p.ListEngineEvents(identifier)
			if err != nil {
				return w.Errorf("failed to list events: %s", err)
			}

			return w.JSON(events)
		})

		r.POST("/events/batch", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := updateIdentifier.Value(r)

			var request apitype.EngineEventBatch
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid batch: %s", err)
			}

			if err := p.AddEngineEvents(identifier, request.Events); err != nil {
				return w.Errorf("failed to write events: %s", err)
			}

			return nil
		})
	}
}
