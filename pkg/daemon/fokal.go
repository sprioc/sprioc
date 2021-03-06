package daemon

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"crypto/rsa"
	"encoding/json"
	"io/ioutil"

	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	newrelic "github.com/newrelic/go-agent"

	"github.com/dgrijalva/jwt-go"
	"github.com/fokal/fokal-core/pkg/conn"
	"github.com/fokal/fokal-core/pkg/handler"
	"github.com/fokal/fokal-core/pkg/logging"
	"github.com/fokal/fokal-core/pkg/routes"
	raven "github.com/getsentry/raven-go"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/rs/cors"
	"github.com/unrolled/secure"
)

type Config struct {
	Port int
	Host string

	Local bool

	PostgresURL        string
	RedisURL           string
	GoogleToken        string
	AWSAccessKeyId     string
	AWSSecretAccessKey string

	SentryURL  string
	NewRelicID string
}

var AppState handler.State

const PublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsW3uHvJvqaaMIW8wKP2E
NI3oVRghsNwUV4VN+5UH2oMAEaYaHiUfOvhXXRjPZo3q8f+v3rS4R7gfJXe8efP0
3x87DRB1uJlNNS777xDISnTLzVAOFFkLOTL9bOTJBlb69yCRhHV1NdUIPCGWntWC
WdKZBJ2zHOQUQgPpAn31imsYlvmlrLEoGNqKOPUQjwdtxEqEYpZyN84Hj5/NIhTC
F6rU8FhReQzEL27BHPfbUwTWUApmtfvCtrSc9pVM3MtlsMOf4OfoGg65kF5HJ/S8
tKRtL24z48ya+ntjbwbE3A5pEswm/Vm19wd77qbY5UILLmNf0xMQfwrkT/IcnBoD
pQIDAQAB
-----END PUBLIC KEY-----`

func Run(cfg *Config) {
	flag := log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	log.SetFlags(flag)

	router := mux.NewRouter()
	api := router.PathPrefix("/v0/").Subrouter()

	log.Printf("Serving at http://%s:%d", cfg.Host, cfg.Port)
	err := raven.SetDSN(cfg.SentryURL)
	if err != nil {
		log.Fatal("Sentry IO not configured")
	}

	config := newrelic.NewConfig("Fokal", cfg.NewRelicID)
	app, err := newrelic.NewApplication(config)
	if err != nil {
		log.Fatal("New Relic not configured")
	}

	if cfg.Local {
		cfg.PostgresURL = cfg.PostgresURL + "?sslmode=disable"
	}

	AppState.Vision, AppState.Maps, _ = conn.DialGoogleServices(cfg.GoogleToken)
	AppState.DB = conn.DialPostgres(cfg.PostgresURL)
	AppState.RD = conn.DialRedis(cfg.RedisURL)
	AppState.Local = cfg.Local
	AppState.Port = cfg.Port
	AppState.DB.SetMaxOpenConns(20)
	AppState.DB.SetMaxIdleConns(50)
	AppState.KeyHash = "554b5db484856bfa16e7da70a427dc4d9989678a"

	// RSA Keys
	AppState.SessionLifetime = time.Hour * 16

	AppState.RefreshAt = time.Minute * 15

	// Refreshing Materialized View
	refreshMaterializedView()

	AppState.PrivateKey, AppState.PublicKeys = ParseKeys()
	refreshGoogleOauthKeys()

	var secureMiddleware = secure.New(secure.Options{
		AllowedHosts:          []string{"api.fok.al", "alpha.fok.al", "beta.fok.al", "fok.al"},
		HostsProxyHeaders:     []string{"X-Forwarded-Host"},
		SSLRedirect:           true,
		SSLHost:               "api.fok.al",
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:            315360000,
		STSIncludeSubdomains:  true,
		STSPreload:            true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self'",
		IsDevelopment:         AppState.Local,
	})

	var crs = cors.New(cors.Options{
		AllowedOrigins:     []string{"https://fok.al", "https://beta.fok.al", "https://alpha.fok.al", "http://localhost:3000"},
		AllowCredentials:   true,
		OptionsPassthrough: true,
		AllowedHeaders:     []string{"Authorization", "Content-Type"},
		AllowedMethods:     []string{"GET", "PUT", "OPTIONS", "PATCH", "POST", "DELETE"},
	})

	var base = alice.New(
		handler.NewRelic(app),
		handler.SentryRecovery,
		//ratelimit.RateLimit,
		crs.Handler,
		handler.Timeout,
		logging.IP, logging.UUID, secureMiddleware.Handler,
		context.ClearHandler, handlers.CompressHandler, logging.ContentTypeJSON)

	//  ROUTES
	routes.RegisterCreateRoutes(&AppState, api, base)
	routes.RegisterModificationRoutes(&AppState, api, base)
	routes.RegisterRetrievalRoutes(&AppState, api, base)
	routes.RegisterSocialRoutes(&AppState, api, base)
	routes.RegisterSearchRoutes(&AppState, api, base)
	routes.RegisterRandomRoutes(&AppState, api, base)
	routes.RegisterAuthRoutes(&AppState, api, base)
	routes.RegisterStatusRoutes(&AppState, api, base)
	api.NotFoundHandler = base.Then(http.HandlerFunc(handler.NotFound))

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.Port),
		handlers.LoggingHandler(os.Stdout, router)))
}

func ParseKeys() (*rsa.PrivateKey, map[string]*rsa.PublicKey) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	keys := make(map[string]string)
	err = json.Unmarshal(body, &keys)
	if err != nil {
		log.Fatal(err)
	}

	parsedKeys := make(map[string]*rsa.PublicKey)

	for kid, pem := range keys {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
		if err != nil {
			log.Fatal(err)
		}
		parsedKeys[kid] = publicKey
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(PublicKey))
	if err != nil {
		log.Fatal(err)
	}
	parsedKeys[AppState.KeyHash] = publicKey

	privateStr := os.Getenv("PRIVATE_KEY")
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateStr))
	if err != nil {
		log.Fatal(err)
	}
	return privateKey, parsedKeys

}

func refreshMaterializedView() {
	tick := time.NewTicker(time.Minute * 10)
	go func() {
		for range tick.C {
			log.Println("Refreshing Materialized View")
			_, err := AppState.DB.Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY searches;")
			if err != nil {
				log.Println(err)
			}
		}
	}()
}

func refreshGoogleOauthKeys() {
	tick := time.NewTicker(time.Minute * 10)
	go func() {
		for range tick.C {
			log.Println("Refreshing Google Auth Keys")
			resp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
			if err != nil {
				log.Fatal(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			resp.Body.Close()

			keys := make(map[string]string)
			err = json.Unmarshal(body, &keys)
			if err != nil {
				log.Fatal(err)
			}
			for kid, pem := range keys {
				publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
				if err != nil {
					log.Fatal(err)
				}
				AppState.PublicKeys[kid] = publicKey
			}
		}
	}()
}
