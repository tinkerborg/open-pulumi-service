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
	"github.com/tinkerborg/open-pulumi-service/internal/handler/api/stacks/stack/update"
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

		r.GET("/", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := stackIdentifier.Value(r)

			stack, err := p.GetStack(identifier)
			if errors.Is(err, store.ErrNotFound) {
				return w.WithStatus(http.StatusNotFound).Errorf("stack not found")
			}

			if err != nil {
				fmt.Printf("ERR %s\n", err)
			}

			return w.JSON(stack)
		})

		r.DELETE("/", func(w *router.ResponseWriter, r *http.Request) error {
			// TODO - delete resources associated with stack
			identifier := stackIdentifier.Value(r)
			return w.Error(p.DeleteStack(identifier))
		})

		r.GET("/export", func(w *router.ResponseWriter, r *http.Request) error {
			identifier := stackIdentifier.Value(r)

			deployment, err := p.GetStackDeployment(identifier)
			// TODO - consistency store/state here
			if errors.Is(err, store.ErrNotFound) {
				return w.WithStatus(http.StatusNotFound).Errorf("update not found")
			}

			return w.JSON(deployment)
		})

		r.POST("/encrypt", func(w *router.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			var request apitype.EncryptValueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid payload: %s", err)
			}

			encrypted, err := c.Encrypt(ctx, request.Plaintext)
			if err != nil {
				return w.Errorf("encryption failed: %s", err)
			}

			return w.JSON(&apitype.EncryptValueResponse{
				Ciphertext: encrypted,
			})
		})

		r.POST("/decrypt", func(w *router.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			buf := new(strings.Builder)
			io.Copy(buf, r.Body)

			// TODO check if this works
			var request apitype.DecryptValueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid payload: %s", err)
			}

			decrypted, err := c.Decrypt(ctx, request.Ciphertext)
			if err != nil {
				return w.Errorf("decryption failed: %s", err)
			}

			return w.JSON(apitype.DecryptValueResponse{
				Plaintext: decrypted,
			})
		})

		r.POST("/batch-decrypt", func(w *router.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			var request apitype.BatchDecryptRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid batch: %s", err)
			}

			plaintexts := map[string][]byte{}

			for _, ciphertext := range request.Ciphertexts {
				key := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
				base64.StdEncoding.Encode(key, ciphertext)

				decrypted, err := c.Decrypt(ctx, ciphertext)
				if err != nil {
					return w.Errorf("decryption failed: %s", err)
				}

				plaintexts[string(key)] = decrypted
			}

			return w.JSON(&apitype.BatchDecryptResponse{
				Plaintexts: plaintexts,
			})
		})

		r.POST("/import", func(w *router.ResponseWriter, r *http.Request) error {
			// TODO - support resource import update
			identifier := client.UpdateIdentifier{
				StackIdentifier: stackIdentifier.Value(r),
				UpdateKind:      apitype.StackImportUpdate,
			}

			// TODO - utility for this
			var request *apitype.UntypedDeployment
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid update: %s", err)
			}

			updateID, err := p.CreateImport(identifier, request)
			if err != nil {
				return w.Errorf("import failed: %s", err)
			}

			return w.JSON(apitype.ImportStackResponse{UpdateID: updateID})
		})

		r.POST("/{updateKind}", func(w *router.ResponseWriter, r *http.Request) error {
			identifier, err := updateIdentifier(stackIdentifier, r)
			if err != nil {
				return w.WithStatus(http.StatusBadRequest).Error(err)
			}

			var request *apitype.UpdateProgramRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid update: %s", err)
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
				return w.Errorf("failed to create update: %s", err)
			}

			return w.JSON(apitype.UpdateProgramResponse{
				UpdateID:         *updateID,
				RequiredPolicies: []apitype.RequiredPolicy{},
			})
		})

		r.GET("/updates/{version}", func(w *router.ResponseWriter, r *http.Request) error {
			version, err := strconv.Atoi(r.PathValue("version"))
			if err != nil {
				return w.WithStatus(http.StatusBadRequest).Error(err)
			}

			identifier := stackIdentifier.Value(r)

			update, err := p.GetStackUpdate(identifier, version)
			if errors.Is(err, store.ErrNotFound) {
				return w.WithStatus(http.StatusNotFound).Errorf("update not found")
			}

			return w.JSON(update)
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
