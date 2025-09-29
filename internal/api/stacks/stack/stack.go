package stack

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/api/stacks/stack/update"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
	"github.com/tinkerborg/open-pulumi-service/pkg/router/middleware"
)

func Setup(p *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		stackIdentifier := middleware.NewDynamicPrefix("/{owner}/{project}/{stack}",
			func(r *http.Request) (client.StackIdentifier, error) {
				owner := r.PathValue("owner")
				project := r.PathValue("project")

				stack, err := tokens.ParseStackName(r.PathValue("stack"))
				if err != nil {
					return client.StackIdentifier{}, err
				}

				return client.StackIdentifier{
					Owner:   owner,
					Project: project,
					Stack:   stack,
				}, nil
			})

		r.Use(stackIdentifier.Middleware)

		r.Mount("/", update.Setup(p, stackIdentifier))

		r.GET("/", func(r *http.Request) (any, error) {
			identifier := stackIdentifier.Value(r)

			stack, err := p.GetStack(identifier)
			if errors.Is(err, store.ErrNotFound) {
				return &apitype.Stack{}, &router.HTTPError{Code: 404, Message: "stack not found"}
			}

			if err != nil {
				fmt.Printf("ERR %s\n", err)
			}

			return stack, err
		})

		r.DELETE("/", func(r *http.Request) (any, error) {
			// TODO - delete resources associated with stack
			identifier := stackIdentifier.Value(r)
			return nil, p.DeleteStack(identifier)
		})

		r.GET("/export", func(r *http.Request) (any, error) {
			identifier := stackIdentifier.Value(r)

			deployment, err := p.GetStackDeployment(identifier)
			// TODO - consistency store/state here
			if errors.Is(err, store.ErrNotFound) {
				return nil, &router.HTTPError{Code: 404, Message: "update not found"}
			}

			return deployment, err
		})

		r.POST("/encrypt", func(r *http.Request) (any, error) {
			ctx := r.Context()

			var request apitype.EncryptValueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid payload"}
			}

			encrypted, err := c.Encrypt(ctx, request.Plaintext)
			if err != nil {
				return nil, err
			}

			return &apitype.EncryptValueResponse{
				Ciphertext: encrypted,
			}, nil
		})

		r.POST("/decrypt", func(r *http.Request) (any, error) {
			ctx := r.Context()

			buf := new(strings.Builder)
			io.Copy(buf, r.Body)

			if false {
				var request apitype.DecryptValueRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, &router.HTTPError{Code: 400, Message: "invalid stack"}
				}

				decrypted, err := c.Decrypt(ctx, request.Ciphertext)
				if err != nil {
					return nil, err
				}

				return apitype.DecryptValueResponse{
					Plaintext: decrypted,
				}, nil
			}
			return nil, nil
		})

		r.POST("/batch-decrypt", func(r *http.Request) (any, error) {
			ctx := r.Context()

			var request apitype.BatchDecryptRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid stack"}
			}

			plaintexts := map[string][]byte{}

			for _, ciphertext := range request.Ciphertexts {
				key := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
				base64.StdEncoding.Encode(key, ciphertext)

				decrypted, err := c.Decrypt(ctx, ciphertext)
				if err != nil {
					return nil, err
				}

				plaintexts[string(key)] = decrypted
			}

			return &apitype.BatchDecryptResponse{
				Plaintexts: plaintexts,
			}, nil
		})

		r.POST("/import", func(r *http.Request) (any, error) {
			// TODO - support resource import update
			identifier := client.UpdateIdentifier{
				StackIdentifier: stackIdentifier.Value(r),
				UpdateKind:      apitype.StackImportUpdate,
			}

			var request *apitype.UntypedDeployment
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid update"}
			}

			updateID, err := p.CreateImport(identifier, request)
			if err != nil {
				return nil, err
			}

			return apitype.ImportStackResponse{UpdateID: updateID}, nil
		})

		r.POST("/{updateKind}", func(r *http.Request) (any, error) {
			identifier, err := updateIdentifier(stackIdentifier, r)
			if err != nil {
				return nil, err
			}

			var request *apitype.UpdateProgramRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, &router.HTTPError{Code: 400, Message: "invalid update"}
			}

			// TODO - figure out what this should actually do - missing fields in updateprogram,
			//        unused fields in updateProgramRequest - options! dryrun!
			updateProgram := &apitype.UpdateProgram{
				Name:    request.Name,
				Runtime: request.Runtime,
				Main:    request.Main,
			}

			updateID, err := p.CreateUpdate(identifier, updateProgram, &request.Options, request.Config, &request.Metadata)
			if err != nil {
				return nil, err
			}

			return apitype.UpdateProgramResponse{UpdateID: *updateID, RequiredPolicies: []apitype.RequiredPolicy{}}, nil
		})

		r.GET("/updates/{version}", func(r *http.Request) (any, error) {
			version, err := strconv.Atoi(r.PathValue("version"))
			if err != nil {
				return nil, err
			}

			// TODO
			identifier := stackIdentifier.Value(r)

			update, err := p.GetStackUpdate(identifier, version)
			if errors.Is(err, store.ErrNotFound) {
				return nil, &router.HTTPError{Code: 404, Message: "update not found"}
			}

			return update, err
		})

	}
}

func updateIdentifier(prefix *middleware.DynamicPrefix[client.StackIdentifier], r *http.Request) (client.UpdateIdentifier, error) {
	updateKind, err := model.ParseUpdateKind(r.PathValue("updateKind"))
	if err != nil {
		return client.UpdateIdentifier{}, err
	}
	updateID := r.PathValue("updateID")

	return client.UpdateIdentifier{
		StackIdentifier: prefix.Value(r),
		UpdateKind:      updateKind,
		UpdateID:        updateID,
	}, nil
}
