package opaqueserver

import "context"

type Storage interface {
	// XXX <28-01-22, afjoseph> For this flow to work, you should expire the
	// UserRequest after X seconds
	StoreUserRequest(ctx context.Context, apptoken string, username string, req *UserRequest) error
	LoadUserRequest(ctx context.Context, apptoken string, username string) (req *UserRequest, err error)
	HasUserRequest(ctx context.Context, apptoken string, username string) (ok bool, err error)

	StoreUserEnvelope(ctx context.Context, apptoken string, username string, env *UserEnvelope) error
	LoadUserEnvelope(ctx context.Context, apptoken string, username string) (env *UserEnvelope, err error)

	StoreAuthNonce(ctx context.Context, apptoken string, username string, nonce []byte) error
	HasAuthNonce(ctx context.Context, apptoken string, username string, nonce []byte) (ok bool, err error)

	// XXX <28-01-22, afjoseph> You must expire the session token every X
	// seconds, however long the client should be able to login without
	// entering a password
	// StoreUserSessionKey(username string, env *UserSessionKey) error
	// LoadUserSessionKey(username string) (env *UserSessionKey, err error)
	// HasUserSessionKey(username string) (ok bool, err error)
}
