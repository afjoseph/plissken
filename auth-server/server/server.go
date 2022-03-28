package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/afjoseph/plissken-auth-server/opaqueserver"
	"github.com/afjoseph/plissken-auth-server/rediswrapper"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type MyServer struct {
	HttpServer *http.Server
	Port       string

	gitCommitHash       string
	sdkVersion          string
	redisWrapper        *rediswrapper.RedisWrapper
	opaqueServer        *opaqueserver.Server
	corsOriginWhitelist []string
}

func Host(
	serverPrivKey []byte,
	corsOriginWhitelist []string,
	addr string,
	verbose bool,
	sdkVersion string,
	gitCommitHash string,
	rdw *rediswrapper.RedisWrapper,
	errChan chan<- error,
) (*MyServer, error) {
	logrus.Tracef("Host with corsOriginWhitelist: %v | addr: %v | verbose: %v",
		corsOriginWhitelist, addr, verbose)

	// Init OpaqueServer
	opaqueServer, err := opaqueserver.NewServer(rdw, serverPrivKey)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	// Init server code
	if verbose {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	gin.DefaultWriter = os.Stdout
	router.Use(
		gin.Logger(),
		gin.Recovery(),
		handleErrors,
		cors.New(
			func() cors.Config {
				cfg := cors.DefaultConfig()
				cfg.AllowHeaders = []string{
					"Origin", "Content-Length", "Content-Type", "Authorization"}
				// TODO <27-02-22, afjoseph> Change this to the actual website
				if corsOriginWhitelist == nil {
					cfg.AllowAllOrigins = true
				} else {
					cfg.AllowOriginFunc = func(origin string) bool {
						logrus.Debugf("Asking for origin: %s", origin)
						for _, v := range corsOriginWhitelist {
							if strings.Contains(origin, v) {
								return true
							}
						}
						return false
					}
				}
				return cfg
			}(),
		),
	)

	// Handlers
	srv := &MyServer{
		opaqueServer:        opaqueServer,
		corsOriginWhitelist: corsOriginWhitelist,
		redisWrapper:        rdw,
		sdkVersion:          sdkVersion,
		gitCommitHash:       gitCommitHash,
	}
	router.GET("/health", func(c *gin.Context) { srv.handleHealthRoute(c) })
	router.POST("/start_password_registration", func(c *gin.Context) {
		srv.handleStartPasswordRegistration(c)
	})
	router.POST("/finalize_password_registration", func(c *gin.Context) {
		srv.handleFinalizePasswordRegistration(c)
	})
	router.POST("/start_password_authentication", func(c *gin.Context) {
		srv.handleStartPasswordAuthentication(c)
	})
	router.POST("/finalize_password_authentication", func(c *gin.Context) {
		srv.handleFinalizePasswordAuthentication(c)
	})
	router.GET("/check-credentials", func(c *gin.Context) {
		srv.handleCheckCredentials(c)
	})
	router.GET("/", func(c *gin.Context) {
		srv.handleIndex(c)
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	srv.HttpServer = &http.Server{
		Addr:    ln.Addr().(*net.TCPAddr).String(),
		Handler: router,
	}
	srv.Port = strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	go func() {
		err := srv.HttpServer.Serve(ln)
		if err != nil && err != http.ErrServerClosed && errChan != nil {
			errChan <- errors.Wrap(err, "")
		}
	}()
	return srv, nil
}

func (s *MyServer) Shutdown(ctx context.Context) error {
	return s.HttpServer.Shutdown(ctx)
}
