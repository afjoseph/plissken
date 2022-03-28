package opaqueserver

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/afjoseph/plissken-auth-server/opaquecommon"
	"github.com/cloudflare/circl/dh/x25519"
	"github.com/cloudflare/circl/oprf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/hkdf"
)

const DefaultAuthNonceLength = 12
const DefaultSessionTokenLength = x25519.Size + DefaultAuthNonceLength

type Server struct {
	storageInterface Storage
	privS, PubS      x25519.Key
}

type UserRequest struct {
	SerializedClientOprvPrivateKey []byte `json:"client_oprf_priv_key"`
}

type UserEnvelope struct {
	PubU                     []byte `json:"user_pub_key"`
	EnvU                     []byte `json:"envu"`
	EnvUNonce                []byte `json:"envu_nonce"`
	RwdUSalt                 []byte `json:"user_key_salt"`
	SerializedOprvPrivateKey []byte `json:"oprf_priv_key"`
}

func NewServer(storageInterface Storage, inputPrivKey []byte) (*Server, error) {
	var privKey, pubKey x25519.Key
	if inputPrivKey == nil {
		_, err := io.ReadFull(cryptoRand.Reader, privKey[:])
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
	} else {
		copy(privKey[:], inputPrivKey)
	}
	x25519.KeyGen(&pubKey, &privKey)
	logrus.Debugf("Using OPAQUE server pubKey: %s", hex.EncodeToString(pubKey[:]))
	return &Server{
		storageInterface: storageInterface,
		PubS:             pubKey,
		privS:            privKey,
	}, nil
}

//    Store request
//    	u.username
//    	u.info
//		u.Ku -> created now
//		  not our concern, but we can expire the request after X seconds
//		  to avoid overloading
//	  And evaluate the OPRF
func (s *Server) HandleNewUserRequest(
	ctx context.Context,
	apptoken, username string,
	evalReq *oprf.EvaluationRequest,
) (*oprf.Evaluation, error) {
	kU, err := oprf.GenerateKey(opaquecommon.OprfSuiteID, cryptoRand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	serializedKu, err := kU.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ret, err := oprf.NewServer(opaquecommon.OprfSuiteID, kU).
		Evaluate(evalReq)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if ret == nil {
		return nil, errors.New("Empty response")
	}

	// TODO <28-01-22, afjoseph> Can think about making this async, but I think
	// it is wiser for state management to keep it sync
	err = s.storageInterface.StoreUserRequest(
		ctx,
		apptoken, username,
		&UserRequest{
			SerializedClientOprvPrivateKey: serializedKu,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return ret, nil
}

func (s *Server) StoreUserData(
	ctx context.Context,
	apptoken, username string,
	pubU, envU, nonce, rwdUSalt []byte) error {
	userReq, err := s.storageInterface.LoadUserRequest(ctx, apptoken, username)
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = s.storageInterface.StoreUserEnvelope(
		ctx,
		apptoken,
		username,
		&UserEnvelope{
			PubU:                     pubU,
			EnvU:                     envU,
			EnvUNonce:                nonce,
			RwdUSalt:                 rwdUSalt,
			SerializedOprvPrivateKey: userReq.SerializedClientOprvPrivateKey,
		},
	)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (s *Server) HandleNewUserAuthentication(
	ctx context.Context,
	apptoken, username string,
	evalReq *oprf.EvaluationRequest,
) (eval *oprf.Evaluation,
	envU, nonce, rwdUSalt, authNonce []byte,
	err error) {
	// Fetch kU from our storage
	savedUserEnv, err := s.storageInterface.LoadUserEnvelope(ctx, apptoken, username)
	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "")
	}
	kU := &oprf.PrivateKey{}
	err = kU.UnmarshalBinary(opaquecommon.OprfSuiteID, savedUserEnv.SerializedOprvPrivateKey)
	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "")
	}
	eval, err = oprf.NewServer(opaquecommon.OprfSuiteID, kU).Evaluate(evalReq)
	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "")
	}
	if eval == nil {
		return nil, nil, nil, nil, nil, errors.New("empty response")
	}

	// AuthRequest creation
	authNonce = make([]byte, DefaultAuthNonceLength)
	_, err = cryptoRand.Read(authNonce)
	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "")
	}
	err = s.storageInterface.StoreAuthNonce(ctx, apptoken, username, authNonce)
	if err != nil {
		// TODO <22-04-2022, afjoseph> Accommodate for duplicate salt errors
		return nil, nil, nil, nil, nil, errors.Wrap(err, "")
	}

	return eval, savedUserEnv.EnvU,
		savedUserEnv.EnvUNonce,
		savedUserEnv.RwdUSalt,
		authNonce, nil
}

func (s *Server) IsRegistered(
	ctx context.Context,
	apptoken, username string,
) (bool, error) {
	return s.storageInterface.HasUserRequest(ctx, apptoken, username)
}

func (s *Server) IsAuthenticated(
	ctx context.Context,
	apptoken, username string,
	inputSessionToken []byte,
) (bool, error) {
	// Check length
	if len(inputSessionToken) != DefaultSessionTokenLength {
		return false, errors.New("bad session token length")
	}

	// Check if auth request exists
	ok, err := s.storageInterface.HasAuthNonce(ctx, apptoken, username,
		inputSessionToken[:DefaultAuthNonceLength])
	if err != nil {
		return false, errors.Wrap(err, "")
	}
	if !ok {
		return false, errors.New("no auth request found")
	}
	authNonce := inputSessionToken[:DefaultAuthNonceLength]

	// Fetch the user envelope
	savedUserEnv, err := s.storageInterface.LoadUserEnvelope(ctx, apptoken, username)
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	// Convert the necessary []byte -> x25519.Key
	var pubU, actualInputSessionToken, sharedKey x25519.Key
	copy(pubU[:], savedUserEnv.PubU)
	copy(actualInputSessionToken[:], inputSessionToken[DefaultAuthNonceLength:])

	// Derive the session key from our side
	ok = x25519.Shared(&sharedKey, &s.privS, &pubU)
	if !ok {
		return false, errors.New("failed to derive session key")
	}

	// Apply authNonce to it
	derivedSessionToken := make([]byte, x25519.Size)
	kdfr := hkdf.New(sha256.New, sharedKey[:], authNonce, nil)
	_, err = io.ReadFull(kdfr, derivedSessionToken)
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	// Compare
	if !bytes.Equal(actualInputSessionToken[:], derivedSessionToken) {
		return false, errors.New("session keys are not equal")
	}
	return true, nil
}
