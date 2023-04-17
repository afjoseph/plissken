package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	plisskenclient "github.com/afjoseph/plissken-protocol/client"
	plisskenserver "github.com/afjoseph/plissken-protocol/server"
	"github.com/alicebob/miniredis/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const testAppToken = "testAppToken"

type testStorageImpl struct {
	r *miniredis.Miniredis
}

func redisKey_UserEnvelope(apptoken, username string) string {
	return fmt.Sprintf("auth:%s:%s:envelope", apptoken, username)
}

func redisKey_UserRequest(apptoken, username string) string {
	return fmt.Sprintf("auth:%s:%s:request", apptoken, username)
}

// Unused but useful to keep here just for reference
// func redisKey_SessionToken(apptoken, username string) string {
// 	return fmt.Sprintf("auth:%s:%s:token", apptoken, username)
// }

func redisKey_AuthNonces(apptoken, username string) string {
	return fmt.Sprintf("auth:%s:%s:requests", apptoken, username)
}

func (s testStorageImpl) StoreUserRequest(ctx context.Context, apptoken, username string, req *plisskenserver.UserRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = s.r.Set(redisKey_UserRequest(apptoken, username), string(b))
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s testStorageImpl) LoadUserRequest(ctx context.Context, apptoken, username string) (*plisskenserver.UserRequest, error) {
	var req plisskenserver.UserRequest
	str, err := s.r.Get(redisKey_UserRequest(apptoken, username))
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

func (s testStorageImpl) HasUserRequest(ctx context.Context, apptoken, username string) (bool, error) {
	return s.r.Exists(redisKey_UserRequest(apptoken, username)), nil
}

func (s testStorageImpl) StoreUserEnvelope(ctx context.Context, apptoken, username string, req *plisskenserver.UserEnvelope) error {
	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "")
	}
	s.r.Set(redisKey_UserEnvelope(apptoken, username), string(b))
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s testStorageImpl) LoadUserEnvelope(ctx context.Context, apptoken, username string) (*plisskenserver.UserEnvelope, error) {
	var req plisskenserver.UserEnvelope
	str, err := s.r.Get(redisKey_UserEnvelope(apptoken, username))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	err = json.Unmarshal([]byte(str), &req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &req, nil
}

func (s testStorageImpl) StoreAuthNonce(
	ctx context.Context,
	apptoken, username string,
	nonce []byte) error {
	_, err := s.r.Lpush(
		redisKey_AuthNonces(apptoken, username),
		hex.EncodeToString(nonce))
	if err != nil {
		return errors.Wrap(err, "")
	}
	fmt.Println(s.r.Dump())

	return nil
}

func (s testStorageImpl) HasAuthNonce(
	ctx context.Context,
	apptoken, username string,
	inputAuthNonce []byte) (bool, error) {
	t, err := s.r.List(redisKey_AuthNonces(apptoken, username))
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

func doPasswordRegistration(ctx context.Context, s *plisskenserver.Server, username, password string) error {
	// 1. Client starts the OPRF process
	_, finData, evalReq, err := plisskenclient.MakeOprfRequest(password)
	if err != nil {
		return err
	}

	// 2. Client -> Server first msg:
	// - the initial parameters to the OPRF function from the client-side

	// 3. Server receives and evaluates OPRF
	// Does two things:
	// a) calls an interface function to store the state of the registration
	//
	//    Store request
	//    	u.username
	//    	u.PubU
	//		u.Ku -> created now
	//		  not our concern, but we can expire the request after X seconds
	//		  to avoid overloading
	//	  And evaluate the OPRF
	sEval, err := s.HandleNewUserRequest(ctx, testAppToken, username, evalReq)
	if err != nil {
		return err
	}

	// 4. Server -> Client: send OPRF evaluation

	// 5. Client receives and finalizes OPRF
	// XXX <27-02-22, afjoseph> You can't "recreate" the OPRF request using the
	// same password here, since it will yield a different result: the function
	// evaluation you and the server negotiated should remain the same.
	// 6. Client hardens OPRF's result: this will be their password
	// 7. Client makes envU, encodes it and encrypts it
	envU, envUNonce, pubU, salt, err := plisskenclient.MakeEnvU(finData, sEval, s.PubS)
	if err != nil {
		return err
	}
	fmt.Printf("envU = %+v\n", envU)

	// 8. Client sends envU, envUNonce, pubU, and salt to server and
	//    deletes everything else

	// 9. Server stores (envU, pubS, pubU, privS, and kU)
	err = s.StoreUserData(ctx, testAppToken, username, pubU, envU,
		envUNonce, salt)
	if err != nil {
		return err
	}
	return nil
}

func doPasswordAuthentication(
	ctx context.Context,
	s *plisskenserver.Server,
	username, password string,
) ([]byte, error) {
	// 1. Client starts the OPRF process
	_, finData, evalReq, err := plisskenclient.MakeOprfRequest(password)
	if err != nil {
		return nil, err
	}

	// 2. Client -> Server first msg:
	// - the initial parameters to the OPRF function from the client-side

	// 3. Server receives and evaluates OPRF
	sAuthEval, envU, envUNonce,
		rwdUSalt, authNonce, err := s.HandleNewUserAuthentication(
		context.Background(), testAppToken, username, evalReq)
	if err != nil {
		return nil, err
	}

	// 4. Server -> Client: send OPRF evaluation

	// 5. Client receives and finalizes OPRF
	// 6. Client hardens OPRF's result: this will be their password
	// 7. Client decrypts envU and derives shared key
	sessionToken, err := plisskenclient.DeriveSessionToken(
		finData,
		sAuthEval,
		envU, envUNonce,
		rwdUSalt,
		authNonce)
	if err != nil {
		return nil, err
	}
	fmt.Printf("sessionToken = %+v\n", sessionToken)
	return sessionToken, nil
}

func TestProtocol(t *testing.T) {
	t.Run("Happy path: register -> login -> access private resource", func(t *testing.T) {
		username := "truebeef"
		password := "bunnyfoofoo"
		s, err := plisskenserver.NewServer(testStorageImpl{miniredis.RunT(t)}, nil)
		require.NoError(t, err)
		fmt.Printf("s = %+v\n", s)
		err = doPasswordRegistration(context.Background(), s, username, password)
		require.NoError(t, err)

		sessionToken, err := doPasswordAuthentication(context.Background(), s, username, password)
		require.NoError(t, err)

		ok, err := s.IsAuthenticated(context.Background(),
			testAppToken, username, sessionToken)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("register -> login with same username but different password should fail", func(t *testing.T) {
		username := "truebeef"
		password := "bunnyfoofoo"
		s, err := plisskenserver.NewServer(testStorageImpl{miniredis.RunT(t)}, nil)
		require.NoError(t, err)
		fmt.Printf("s = %+v\n", s)
		err = doPasswordRegistration(context.Background(), s, username, password)
		require.NoError(t, err)

		password = "notbunnyfoofoo"
		_, err = doPasswordAuthentication(context.Background(), s, username, password)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "message authentication failed")
	})

	t.Run("multiple registrations with the same credentials should yield different session tokens", func(t *testing.T) {
		username := "truebeef"
		password := "bunnyfoofoo"
		s, err := plisskenserver.NewServer(testStorageImpl{miniredis.RunT(t)}, nil)
		require.NoError(t, err)

		err = doPasswordRegistration(context.Background(), s, username, password)
		require.NoError(t, err)
		sessionToken1, err := doPasswordAuthentication(context.Background(), s, username, password)
		require.NoError(t, err)
		require.NotNil(t, sessionToken1)

		err = doPasswordRegistration(context.Background(), s, username, password)
		require.NoError(t, err)
		sessionToken2, err := doPasswordAuthentication(context.Background(), s, username, password)
		require.NoError(t, err)
		require.NotNil(t, sessionToken2)

		require.NotEqual(t, sessionToken1[:], sessionToken2[:])
	})

	t.Run("multiple logins with the same credentials should yield different session tokens", func(t *testing.T) {
		username := "truebeef"
		password := "bunnyfoofoo"
		s, err := plisskenserver.NewServer(testStorageImpl{miniredis.RunT(t)}, nil)
		require.NoError(t, err)

		err = doPasswordRegistration(context.Background(), s, username, password)
		require.NoError(t, err)

		sessionToken1, err := doPasswordAuthentication(context.Background(), s, username, password)
		require.NoError(t, err)
		require.NotNil(t, sessionToken1)

		sessionToken2, err := doPasswordAuthentication(context.Background(), s, username, password)
		require.NoError(t, err)
		require.NotNil(t, sessionToken2)

		require.NotEqual(t, sessionToken1[:], sessionToken2[:])
	})
}
