package main

import (
	"log"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/tinkerborg/open-pulumi-service/internal/api"
	"github.com/tinkerborg/open-pulumi-service/internal/service/crypto"
	"github.com/tinkerborg/open-pulumi-service/internal/service/state"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/pkg/router"
	"github.com/tinkerborg/open-pulumi-service/pkg/router/middleware"
)

// TODO - break this up and let services define their own config
// support multiple flavors of services e.g. google KMS, aws KMS etc
// currently postgres and GCP KMS are the only options, so they're required
type Config struct {
	GoogleKeyID   string `env:"GCP_KMS_KEY_ID,required"`
	DatabaseURL   string `env:"DATABASE_URL,required"`
	ListenAddress string `env:"LISTEN_ADDRESS" envDefault:"0.0.0.0"`
	ListenPort    string `env:"LISTEN_PORT" envDefault:"8080"`
}

func main() {
	config := Config{}
	if err := env.Parse(&config); err != nil {
		log.Fatalf("error parsing configuration: %s", err)
	}

	s, err := store.NewPostgres(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Couldn't instantiate store: %s", err)
	}

	stateService := state.New(s)

	cryptoService := crypto.NewGoogleKmsCryptoService(config.GoogleKeyID)

	r := router.NewRouter()

	r.Use(middleware.Logging, middleware.GzipDecode)

	r.Mount("/api", api.Setup(stateService, cryptoService))

	log.Fatal(http.ListenAndServe(config.ListenAddress+":"+config.ListenPort, r))
}
