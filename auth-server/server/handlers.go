package server

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	plisskencommon "github.com/afjoseph/plissken-protocol/common"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const defaultExpiryDuration = 15 * time.Minute

func handleErrors(c *gin.Context) {
	c.Next() // execute all the handlers
	for _, appErr := range c.Errors {
		logrus.Errorf("Error occurred: %v", appErr.Err)
	}
	errorToPrint := c.Errors.ByType(gin.ErrorTypePublic).Last()
	if errorToPrint != nil && errorToPrint.Meta != nil {
		if c.Writer.Written() {
			fmt.Fprintf(c.Writer, "Error: %s\n", errorToPrint.Meta)
		} else {
			c.String(500, errorToPrint.Meta.(string))
		}
	}
}

func (s *MyServer) handleHealthRoute(c *gin.Context) {
	c.String(200, s.gitCommitHash+":"+s.sdkVersion)
	c.Status(http.StatusOK)
}

func (s *MyServer) handleStartPasswordRegistration(c *gin.Context) {
	b, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		return
	}
	logrus.Debugf("%+v", string(b))

	var req plisskencommon.OprfRequestResults
	err = c.MustBindWith(&req, binding.JSON)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("JSON body is bad")
		return
	}

	ok, err := s.opaqueServer.IsRegistered(
		c.Request.Context(), req.AppToken, req.Username)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("request failed to evaluate")
		return
	}
	if ok {
		c.String(http.StatusForbidden, "User already registered")
		return
	}

	eval, err := s.opaqueServer.HandleNewUserRequest(
		c.Request.Context(), req.AppToken, req.Username, req.EvalReq)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("request failed to evaluate")
		return
	}

	c.JSON(200, &plisskencommon.OprfServerEvaluation{Eval: eval})
}

func (s *MyServer) handleFinalizePasswordRegistration(c *gin.Context) {
	b, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		return
	}
	logrus.Debugf("%+v", string(b))

	var req plisskencommon.PasswordRegistrationData
	err = c.MustBindWith(&req, binding.JSON)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("JSON body is bad")
		return
	}

	err = s.opaqueServer.StoreUserData(
		c.Request.Context(),
		req.AppToken, req.Username, req.PubU, req.EnvU,
		req.EnvUNonce, req.Salt,
	)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("Failed to store user data")
		return
	}
	c.Status(200)
}

func (s *MyServer) handleStartPasswordAuthentication(c *gin.Context) {
	b, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		return
	}
	logrus.Debugf("%+v", string(b))

	var req plisskencommon.OprfRequestResults
	err = c.MustBindWith(&req, binding.JSON)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("JSON body is bad")
		return
	}

	eval, envU,
		envUNonce,
		rwdUSalt, authNonce, err := s.opaqueServer.HandleNewUserAuthentication(
		c.Request.Context(), req.AppToken,
		req.Username, req.EvalReq)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("request failed to evaluate")
		return
	}
	c.JSON(200, &plisskencommon.StartPasswordAuthServerResp{
		Eval:      eval,
		EnvU:      envU,
		EnvUNonce: envUNonce,
		RwdUSalt:  rwdUSalt,
		AuthNonce: authNonce,
	})
}

func (s *MyServer) handleFinalizePasswordAuthentication(c *gin.Context) {
	var req plisskencommon.FinalizePasswordAuthData
	err := c.MustBindWith(&req, binding.JSON)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic)
		return
	}

	// Decode session token and check it
	b, err := hex.DecodeString(req.SessionToken)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("session token is bad")
		return
	}
	ok, err := s.opaqueServer.IsAuthenticated(
		c.Request.Context(),
		req.AppToken,
		req.Username, b)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("while checking session token")
		return
	}
	if !ok {
		c.String(http.StatusUnauthorized, "Session token is invalid")
		return
	}

	// Session token is valid: store it for future use
	err = s.redisWrapper.StoreSessionToken(
		c.Request.Context(),
		req.AppToken, req.Username, req.SessionToken,
		defaultExpiryDuration,
	)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("while saving session token")
		return
	}

	c.Status(http.StatusOK)
}

type CheckCredentialsRequestData struct {
	AppToken     string `form:"apptoken"`
	AppSecret    string `form:"appsecret"`
	Username     string `form:"username"`
	SessionToken string `form:"session_token"`
}

type CheckCredentialsResponseData struct {
	Username   string `json:"username"`
	CreatedAt  int64  `json:"created_at"`
	SdkVersion string `json:"sdk_version"`
	ExpiresAt  int64  `json:"expires_at"`
}

func (s *MyServer) handleCheckCredentials(c *gin.Context) {
	var req CheckCredentialsRequestData
	err := c.MustBindWith(&req, binding.Query)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic)
		return
	}
	fmt.Printf("req = %+v\n", req)

	// Check app secret
	ok, err := s.redisWrapper.HasAppSecret(
		c.Request.Context(), req.AppToken, req.AppSecret)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("while checking app secret")
		return
	}
	if !ok {
		c.String(http.StatusUnauthorized, "App secret is invalid")
		return
	}

	// Check session token
	ok, err = s.redisWrapper.HasSessionToken(
		c.Request.Context(),
		req.AppToken, req.Username, req.SessionToken,
	)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("while checking app secret")
		return
	}
	if !ok {
		c.String(http.StatusUnauthorized, "Session token is invalid")
		return
	}

	typedResp := CheckCredentialsResponseData{
		Username:   req.Username,
		CreatedAt:  time.Now().Unix(),
		SdkVersion: s.sdkVersion,
		ExpiresAt:  time.Now().Add(defaultExpiryDuration).Unix(),
	}
	c.JSON(http.StatusOK, typedResp)
}

func (s *MyServer) handleIndex(c *gin.Context) {
	var sb strings.Builder

	tokens, err := s.redisWrapper.GetAllAppTokens(c.Request.Context())
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.Wrapf(err, "")).
			SetType(gin.ErrorTypePublic).
			SetMeta("while fetching app tokens")
		return
	}

	for tokIdx := 0; tokIdx < len(tokens); tokIdx++ {
		token := tokens[tokIdx]
		if tokIdx > 0 {
			sb.WriteString("======================================================\n")
			sb.WriteString("======================================================\n")
		}
		usernames, err := s.redisWrapper.GetAllUsernamesFromEnvelopes(
			c.Request.Context(), token)
		if err != nil {
			c.AbortWithError(
				http.StatusInternalServerError,
				errors.Wrapf(err, "")).
				SetType(gin.ErrorTypePublic).
				SetMeta(fmt.Sprintf("while fetching usernames for app token %s", token))
			return
		}

		for usrIdx := 0; usrIdx < len(usernames); usrIdx++ {
			username := usernames[usrIdx]
			if usrIdx > 0 {
				sb.WriteString("---------------------------------------\n")
			}
			env, err := s.redisWrapper.LoadUserEnvelope(c.Request.Context(), token, username)
			if err != nil {
				c.AbortWithError(
					http.StatusInternalServerError,
					errors.Wrapf(err, "")).
					SetType(gin.ErrorTypePublic).
					SetMeta("while checking fetching user envelopes")
				return
			}
			sb.WriteString(fmt.Sprintf("Data for username %s | apptoken %s\n\n", username, token))
			sb.WriteString(fmt.Sprintf("- PubU: %s\n", hex.EncodeToString(env.PubU)))
			sb.WriteString(fmt.Sprintf("- EnvU: %s\n", hex.EncodeToString(env.EnvU)))
			sb.WriteString(fmt.Sprintf("- EnvUNonce: %s\n", hex.EncodeToString(env.EnvUNonce)))
			sb.WriteString(fmt.Sprintf("- RwdUSalt: %s\n", hex.EncodeToString(env.RwdUSalt)))
			sb.WriteString(fmt.Sprintf("- OprfPrivKey: %s\n", hex.EncodeToString(env.SerializedOprvPrivateKey)))
		}
	}
	c.String(http.StatusOK, sb.String())
}
