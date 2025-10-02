package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-pkgz/auth/token"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/service/auth"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

// TODO - TEMPORARY - quick hack - move this to auth service and make provider agnostic
type OAuthConfig struct {
	ClientID   string `env:"CLIENT_ID,required"`
	Secret     string `env:"SECRET,required"`
	AppBaseURL string
}

func Setup(a *auth.Service, s *state.Service, c crypto.Service, config OAuthConfig) router.Setup {
	config.AppBaseURL = strings.TrimSuffix(config.AppBaseURL, "/")

	return func(r *router.Router) {
		provider := github.New(
			config.ClientID,
			config.Secret,
			config.AppBaseURL+"/auth/github/callback",
			"user",
			"read:org",
		)

		key := "your-session-secret" // Replace with secure key
		sessionStore := sessions.NewCookieStore([]byte(key))
		sessionStore.MaxAge(86400 * 30)
		sessionStore.Options.Path = "/"
		sessionStore.Options.HttpOnly = true
		sessionStore.Options.Secure = false // Set to true for HTTPS
		gothic.Store = sessionStore

		r.GET("/auth/github/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			query := r.URL.Query()
			port := query.Get("port")
			nonce := query.Get("nonce")

			state := port + "|" + nonce

			session, err := provider.BeginAuth(state)
			if err != nil {
				return w.Error(err)
			}

			url, err := session.GetAuthURL()
			if err != nil {
				return w.Error(err)
			}

			http.Redirect(w, r, url, http.StatusTemporaryRedirect)

			return nil
		})

		r.GET("/auth/github/callback/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			state := r.URL.Query().Get("state")

			// if state != "user123|home|randomNonce" {
			// return w.WithStatus(http.StatusBadRequest).Errorf("invalid state")
			// }

			parts := strings.Split(state, "|")

			port, nonce := parts[0], parts[1]

			session, err := provider.BeginAuth(state) // Create session
			if err != nil {
				return w.Error(err)
			}

			_, err = session.Authorize(provider, r.URL.Query()) // Process code
			if err != nil {
				return w.Error(err)
			}

			sessionUser, err := provider.FetchUser(session)
			if err != nil {
				return w.Error(err)
			}

			// TODO - map provider attributes
			// TODO - make more generic get or create logic
			user, err := s.GetUserByName(sessionUser.NickName)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					user = createServiceUser(sessionUser)
					if err := s.CreateUser(user); err != nil {
						if !errors.Is(err, store.ErrExist) {
							return w.Error(err)
						}
					}
				}
			}

			token, err := a.CreateToken(user.ID, "token")
			if err != nil {
				return w.Error(err)
			}

			dest := fmt.Sprintf("http://localhost:%s/?accessToken=%s&nonce=%s", port, token, nonce)

			http.Redirect(w, r, dest, http.StatusFound)

			return nil
		})

		r.GET("/auth/{provider}/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			query := r.URL.Query()
			nonce := query.Get("nonce")
			port := query.Get("port")

			session, _ := gothic.Store.Get(r, "gothic-session")
			session.Values["nonce"] = nonce
			session.Values["port"] = port
			session.Store().Save(r, w, session)

			gothic.BeginAuthHandler(w, r)
			return nil
		})

		r.GET("/welcome/cli/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			session, _ := gothic.Store.Get(r, "gothic-session")
			// TODO - getting wrong session values when switching github account
			w.Write([]byte("Welcome, " + session.Values["email"].(string) + "!"))
			return nil
		})

		r.GET("/auth/{provider}/callback/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			user, err := gothic.CompleteUserAuth(w, r)
			if err != nil {
				return w.Error(err)
			}

			session, _ := gothic.Store.Get(r, "gothic-session")

			session.Values["email"] = user.Email
			session.Store().Save(r, w, session)

			dest := fmt.Sprintf("http://localhost:%s/?accessToken=%s&nonce=%s", session.Values["port"], "oooomlaut", session.Values["nonce"])

			http.Redirect(w, r, dest, http.StatusFound)

			return nil
			// t, _ := template.ParseFiles("templates/success.html")
			// return t.Execute(w, user)
		})

		r.GET("/cli-login/{$}", func(w *router.ResponseWriter, r *http.Request) error {
			query := r.URL.Query()
			nonce := query.Get("cliSessionNonce")
			port := query.Get("cliSessionPort")

			if nonce == "" && port == "" {
				return w.WithStatus(http.StatusBadRequest).Errorf("invalid request")
			}

			dest := fmt.Sprintf("/auth/github?port=%s&nonce=%s", port, nonce)

			http.Redirect(w, r, dest, http.StatusFound)

			gothic.BeginAuthHandler(w, r)

			return w.JSON(struct {
				Token string `json:"token"`
			}{Token: "moo"})
		})
	}
}

func checkGoogleUser(userID, accessToken string) bool {
	ctx := context.Background()
	conf := &oauth2.Config{Endpoint: googleoauth.Endpoint}
	client := conf.Client(ctx, &oauth2.Token{AccessToken: accessToken})
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	return true
}
func claimsUpdaterWithDB() token.ClaimsUpdFunc {
	return func(claims token.Claims) token.Claims {
		if claims.User == nil {
			return claims
		}

		// db := getDB()
		// if !db.UserExists(claims.User.ID) {
		// db.CreateUser(*claims.User)
		// }

		return claims
	}
}

// TODO - map provider attributes
func createServiceUser(user goth.User) *model.ServiceUser {
	var name string
	if user.FirstName != "" && user.LastName != "" {
		name = user.FirstName + " " + user.LastName
	} else if user.FirstName != "" {
		name = user.FirstName
	} else {
		name = user.Name
	}

	return &model.ServiceUser{
		GitHubLogin:   user.NickName,
		Name:          name,
		Email:         user.Email,
		AvatarURL:     user.AvatarURL,
		Organizations: []model.ServiceUserInfo{},
		Identities:    []string{},
		SiteAdmin:     nil,
		TokenInfo:     nil,
	}
}
