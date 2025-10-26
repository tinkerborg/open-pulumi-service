package stack

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/handler/api/stacks/stack/update"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/service/auth"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/util"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
	"github.com/tinkerborg/open-pulumi-service/pkg/router/middleware"
)

func Setup(a *auth.Service, s *state.Service, c crypto.Service) router.Setup {
	return func(r *router.Router) {
		r.WithPrefix("/{owner}/{project}/{stack}", StackIdentifier.Middleware).Do(func(r *router.Router) {
			r.Mount("/", update.Setup(a, s, StackIdentifier))

			r.GET("/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)

				stack, err := s.GetStack(identifier)
				if err != nil {
					return w.Error(err)
				}

				return w.JSON(stack)
			})

			r.DELETE("/", func(w *router.ResponseWriter, r *http.Request) error {
				// TODO - delete resources associated with stack
				identifier := StackIdentifier.Value(r)
				if err := s.DeleteStack(identifier); err != nil {
					return w.Error(err)
				}
				w.Write([]byte{})
				return nil
			})

			r.GET("/resources/{version}/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)
				version := r.PathValue("version")

				stack, err := s.GetStack(identifier)
				if err != nil {
					return w.Error(err)
				}

				versionNumber, err := s.ParseStackVersion(stack, version)
				if err != nil {
					return w.Error(err)
				}

				update, err := s.GetStackUpdate(identifier, version)
				if err != nil {
					return w.Error(err)
				}

				resources, err := s.ListStackResources(client.UpdateIdentifier{UpdateID: update.UpdateID})
				if err != nil {
					return w.Error(err)
				}

				return w.JSON(&ListStackResourcesResponse{
					Resources: resources,
					Version:   versionNumber,
				})
			})

			r.GET("/export/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)

				deployment, err := s.GetStackDeployment(identifier)
				if err != nil {
					return w.Error(err)
				}

				return w.JSON(deployment)
			})

			r.POST("/encrypt/{$}", func(w *router.ResponseWriter, r *http.Request) error {
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

			r.POST("/decrypt/{$}", func(w *router.ResponseWriter, r *http.Request) error {
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

			r.POST("/batch-decrypt/{$}", func(w *router.ResponseWriter, r *http.Request) error {
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

			r.POST("/import/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				// TODO - support resource import update
				identifier := client.UpdateIdentifier{
					StackIdentifier: StackIdentifier.Value(r),
					UpdateKind:      apitype.StackImportUpdate,
				}

				// TODO - utility for this
				var request *apitype.UntypedDeployment
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return w.WithStatus(http.StatusBadRequest).Errorf("invalid update: %s", err)
				}

				updateID, err := s.CreateImport(identifier, request)
				if err != nil {
					return w.Errorf("import failed: %s", err)
				}

				return w.JSON(apitype.ImportStackResponse{UpdateID: updateID})
			})

			r.POST("/{updateKind}/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier, err := updateIdentifier(StackIdentifier, r)
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

				claim, err := a.GetRequestClaims(r)
				if err != nil {
					return w.Error(err)
				}

				user, err := s.GetUser(claim.ID)
				if err != nil {
					return w.Error(err)
				}

				updateID, err := s.CreateUpdate(identifier, updateProgram, &request.Options, request.Config, &request.Metadata, user)
				if err != nil {
					return w.Errorf("failed to create update: %s", err)
				}

				return w.JSON(apitype.UpdateProgramResponse{
					UpdateID:         *updateID,
					RequiredPolicies: []apitype.RequiredPolicy{},
				})
			})

			r.GET("/activity/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)

				pageSize, err := util.IntegerParam(r, "pageSize", 10)
				if err != nil {
					return w.Error(err)
				}

				page, err := util.IntegerParam(r, "page", 1)
				if err != nil {
					return w.Error(err)
				}

				updates, err := s.ListUpdates(identifier, state.ListUpdateOptions{Descending: true, PageSize: pageSize, Page: page})
				if err != nil {
					return w.Error(err)
				}

				activity := []StackActivity{}

				for _, update := range updates {
					activity = append(activity, StackActivity{Update: update})
				}

				count, err := s.GetUpdatesCount(identifier)
				if err != nil {
					return w.Error(err)
				}

				return w.JSON(ListActivityRepsonse{
					Activity:     activity,
					ItemsPerPage: pageSize,
					Total:        count,
				})
			})

			r.GET("/updates/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)

				outputType := r.URL.Query().Get("output-type")
				if outputType != "" && outputType != "cli" && outputType != "service" {
					return w.Errorf("invalid output type '%s'", outputType)
				}

				pageSize, err := util.IntegerParam(r, "pageSize", 10)
				if err != nil {
					return w.Error(err)
				}

				page, err := util.IntegerParam(r, "page", 1)
				if err != nil {
					return w.Error(err)
				}

				updates, err := s.ListUpdates(identifier, state.ListUpdateOptions{PageSize: pageSize, Page: page})
				if err != nil {
					return w.Error(err)
				}

				if outputType == "service" {
					count, err := s.GetUpdatesCount(identifier)
					if err != nil {
						return w.Error(err)
					}

					return w.JSON(&ListPaginatedUpdatesResponse{
						Updates:      updates,
						ItemsPerPage: pageSize,
						Total:        count,
					})
				}

				infos := []apitype.UpdateInfo{}

				for _, update := range updates {
					infos = append(infos, update.Info)
				}

				return w.JSON(&ListUpdatesResponse{Updates: infos})
			})

			r.GET("/updates/{version}/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)

				version := r.PathValue("version")

				update, err := s.GetStackUpdate(identifier, version)
				if err != nil {
					return w.Error(err)
				}

				return w.JSON(update)
			})

			// TODO - why no resourceChanges field?
			r.GET("/updates/{version}/previews/{$}", func(w *router.ResponseWriter, r *http.Request) error {
				identifier := StackIdentifier.Value(r)
				version := r.PathValue("version")

				pageSize, err := util.IntegerParam(r, "pageSize", 0)
				if err != nil {
					return w.Error(err)
				}

				page, err := util.IntegerParam(r, "page", 1)
				if err != nil {
					return w.Error(err)
				}

				previews, err := s.ListPreviews(identifier, version, state.ListUpdateOptions{PageSize: pageSize, Page: page})
				if err != nil {
					return w.Error(err)
				}

				count, err := s.GetPreviewsCount(identifier, version)
				if err != nil {
					return w.Error(err)
				}

				// TODO pagination support
				return w.JSON(&ListPaginatedUpdatesResponse{
					Updates:      previews,
					ItemsPerPage: pageSize,
					Total:        count,
				})
			})
		})
	}
}

var StackIdentifier = middleware.NewPathParser(
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

func updateIdentifier(prefix *middleware.PathParser[client.StackIdentifier], r *http.Request) (client.UpdateIdentifier, error) {
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

type ListStackResourcesResponse struct {
	Resources []apitype.ResourceV3 `json:"resources"`
	Region    string               `json:"region"`
	Version   int                  `json:"version"`
}

type ListPaginatedUpdatesResponse struct {
	Updates      []model.StackUpdate `json:"updates"`
	ItemsPerPage int                 `json:"itemsPerPage"`
	Total        int64               `json:"total"`
}

type ListUpdatesResponse struct {
	Updates []apitype.UpdateInfo `json:"updates"`
}

type ListActivityRepsonse struct {
	Activity     []StackActivity `json:"activity"`
	ItemsPerPage int             `json:"itemsPerPage"`
	Total        int64           `json:"total"`
}

type StackActivity struct {
	Update model.StackUpdate `json:"update"`
}
