package rediswrapper

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	mathRand "math/rand"
	"strings"
	"time"

	plisskenserver "github.com/afjoseph/plissken-protocol/server"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const authNoncesMaxCount = 20

// RedisWrapper implements the plisskenserver.Storage interface
type RedisWrapper struct {
	*redis.Client
}

func redisKey_UserEnvelope(apptoken, username string) string {
	return fmt.Sprintf("reg:%s:%s:envelope", apptoken, username)
}

func redisKey_UserRequest(apptoken, username string) string {
	return fmt.Sprintf("reg:%s:%s:request", apptoken, username)
}

func redisKey_AuthNonces(apptoken, username string) string {
	return fmt.Sprintf("auth:%s:%s:requests", apptoken, username)
}

func redisKey_SessionToken(apptoken, username string) string {
	return fmt.Sprintf("tokens:%s:%s:token", apptoken, username)
}

func redisKey_AppSecret(apptoken string) string {
	return fmt.Sprintf("app_secrets:%s:secret", apptoken)
}

func (s RedisWrapper) StoreUserRequest(
	ctx context.Context,
	apptoken, username string,
	req *plisskenserver.UserRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = s.Set(ctx, redisKey_UserRequest(apptoken, username), string(b), 0).Err()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s RedisWrapper) LoadUserRequest(ctx context.Context, apptoken, username string) (
	*plisskenserver.UserRequest, error) {
	var req plisskenserver.UserRequest
	str, err := s.Get(ctx, redisKey_UserRequest(apptoken, username)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if str == "" {
		return nil, errors.New("saved data is nil")
	}
	err = json.Unmarshal([]byte(str), &req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &req, nil
}

func (s RedisWrapper) HasUserRequest(ctx context.Context, apptoken, username string) (bool, error) {
	n, err := s.Exists(ctx, redisKey_UserRequest(apptoken, username)).Result()
	if err != nil {
		return false, errors.Wrap(err, "")
	}
	return n != 0, nil
}

func (s RedisWrapper) StoreUserEnvelope(
	ctx context.Context,
	apptoken, username string,
	req *plisskenserver.UserEnvelope) error {
	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = s.Set(ctx, redisKey_UserEnvelope(apptoken, username), string(b), 0).Err()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s RedisWrapper) LoadUserEnvelope(
	ctx context.Context,
	apptoken, username string) (*plisskenserver.UserEnvelope, error) {
	var req plisskenserver.UserEnvelope
	str, err := s.Get(ctx, redisKey_UserEnvelope(apptoken, username)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	err = json.Unmarshal([]byte(str), &req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &req, nil
}

func (s RedisWrapper) StoreSessionToken(
	ctx context.Context,
	apptoken, username, sessionToken string,
	expiresAt time.Duration,
) error {
	err := s.Set(ctx, redisKey_SessionToken(apptoken, username), sessionToken, expiresAt).Err()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s RedisWrapper) HasSessionToken(
	ctx context.Context,
	apptoken, username, sessionToken string) (bool, error) {
	t, err := s.Get(ctx, redisKey_SessionToken(apptoken, username)).Result()
	if err != nil {
		return false, errors.Wrap(err, "")
	}
	return t == sessionToken, nil
}

func (s RedisWrapper) StoreAppSecret(
	ctx context.Context,
	apptoken, appSecret string,
) error {
	err := s.Set(ctx, redisKey_AppSecret(apptoken), appSecret, 0).Err()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s RedisWrapper) HasAppSecret(
	ctx context.Context,
	apptoken, appSecret string) (bool, error) {
	t, err := s.Get(ctx, redisKey_AppSecret(apptoken)).Result()
	if err != nil {
		return false, errors.Wrap(err, "")
	}
	return t == appSecret, nil
}

func (s RedisWrapper) StoreAuthNonce(
	ctx context.Context,
	apptoken, username string,
	nonce []byte) error {
	err := s.LPush(ctx, redisKey_AuthNonces(apptoken, username),
		hex.EncodeToString(nonce)).Err()
	if err != nil {
		return errors.Wrap(err, "")
	}
	// Keep only the last 'authNoncesMaxCount' requests, but don't run LTRIM
	// every time; do it on a random chance
	if mathRand.Intn(3) == 0 {
		if err = s.LTrim(ctx, redisKey_AuthNonces(apptoken, username),
			0, authNoncesMaxCount).Err(); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func (s RedisWrapper) HasAuthNonce(
	ctx context.Context,
	apptoken, username string,
	inputAuthNonce []byte) (bool, error) {
	t, err := s.LRange(ctx,
		redisKey_AuthNonces(apptoken, username),
		0, authNoncesMaxCount).Result()
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	for _, v := range t {
		b, err := hex.DecodeString(v)
		if err != nil {
			// TODO <22-04-2022, afjoseph> Deal with corruption
			logrus.Errorf("Failed to decode auth nonce: %s", v)
		}
		if bytes.Equal(b, inputAuthNonce) {
			return true, nil
		}
	}

	return false, nil
}

func (s RedisWrapper) GetAllAppTokens(ctx context.Context) ([]string, error) {
	keys, err := s.Keys(ctx, redisKey_AppSecret("*")).Result()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var tokens []string
	for _, key := range keys {
		arr := strings.Split(key, ":")
		tokens = append(tokens, arr[1])
	}
	return tokens, nil
}

func (s RedisWrapper) GetAllUsernamesFromEnvelopes(
	ctx context.Context, apptoken string) ([]string, error) {
	keys, err := s.Keys(ctx, redisKey_UserEnvelope(apptoken, "*")).Result()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var usernames []string
	for _, key := range keys {
		arr := strings.Split(key, ":")
		usernames = append(usernames, arr[2])
	}
	return usernames, nil
}
