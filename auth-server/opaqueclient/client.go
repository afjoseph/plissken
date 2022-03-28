package opaqueclient

import (
	"io"

	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"crypto/sha256"

	"github.com/afjoseph/plissken-auth-server/opaquecommon"
	"github.com/cloudflare/circl/dh/x25519"
	"github.com/cloudflare/circl/oprf"
	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

func MakeOprfRequest(password string) (
	// Used to recreate the OPRF request client-side when passing it
	// back-and-forth to GopherJS. This means that the server-side doesn't need
	// this.
	inputs [][]byte,
	// Used, along with evalReq, as the OPRF request results
	finData *oprf.FinalizeData,
	evalReq *oprf.EvaluationRequest,
	err error) {
	e := opaquecommon.G.HashToElement(
		[]byte(password),
		// TODO <13-02-22, afjoseph> Document this better
		[]byte("QUUX-V01-CS02-with-P256_XMD:SHA-256_SSWU_RO_"))
	b, err := e.MarshalBinary()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "")
	}
	finData, evalReq, err = oprf.NewClient(opaquecommon.OprfSuiteID).Blind([][]byte{b})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "")
	}
	return [][]byte{b}, finData, evalReq, nil
}

func finalizeRequest(
	finData *oprf.FinalizeData,
	eval *oprf.Evaluation,
) ([][]byte, error) {
	b, err := oprf.NewClient(opaquecommon.OprfSuiteID).Finalize(finData, eval)
	if err != nil {
		return nil, errors.Wrap(err, "oprf.NewClient")
	}
	if b == nil {
		return nil, errors.New("Empty response")
	}
	return b, nil
}

func hardenOprfResult(
	x []byte, precomputedSalt []byte,
) (rwdU []byte, salt []byte, err error) {
	if precomputedSalt != nil {
		salt = precomputedSalt
	} else {
		salt = make([]byte, 32)
		_, err = cryptoRand.Read(salt)
		if err != nil {
			return nil, nil, errors.Wrap(err, "")
		}
	}
	key := argon2.IDKey(x, salt,
		1,
		// TODO <15-04-2022, afjoseph> This is significantly less secure than
		// the amount we need in a production context (something around
		// 64*1024). We're doing it like this since the GopherJS generated code
		// is insanely slow on JS (takes about 10 seconds). Later, we'll
		// implement Argon2 in JS and it'll be much faster.
		128,
		4, 32)
	if key == nil || len(key) == 0 {
		return nil, nil, errors.New("Key is nil or empty")
	}
	return key, salt, nil
}

// Construction of the encrypted envU is:
// ciphertext + tag + nonce
func MakeEnvU(
	finData *oprf.FinalizeData,
	eval *oprf.Evaluation,
	pubS x25519.Key,
) (envU,
	envUNonce,
	pubU,
	salt []byte,
	err error) {
	oprfRet, err := finalizeRequest(finData, eval)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "finalizeRequest")
	}
	rwdU, salt, err := hardenOprfResult(oprfRet[0], nil)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "hardenOprfResult")
	}

	var privU, pubUAsKey x25519.Key
	_, err = io.ReadFull(cryptoRand.Reader, privU[:])
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "")
	}
	x25519.KeyGen(&pubUAsKey, &privU)

	// Encoding envU
	encodedEnvU := []byte{}
	encodedEnvU = append(encodedEnvU, privU[:]...)
	encodedEnvU = append(encodedEnvU, pubS[:]...)

	// Key derivation
	kdfr := hkdf.New(sha256.New, rwdU, nil, nil)
	// XXX <25-01-22, afjoseph> hkdf (hmac-based extract-and-expand key
	// derivation function) is a function that provides a stream, based on a
	// not-so-secure password (called 'key' here), that we can generate secure
	// keys from (like cbcKey and hmacKey)
	aesKey := make([]byte, 16) // AES128-GCM
	_, err = io.ReadFull(kdfr, aesKey)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "io.readfull")
	}

	// AES128-GCM encryption
	ciph, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "aes.newcipher")
	}
	nonce := make([]byte, 12)
	_, err = cryptoRand.Read(nonce)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "cryptorand.read")
	}
	aesGcm, err := cipher.NewGCM(ciph)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "cipher.newgcm")
	}
	envU = aesGcm.Seal(nil, nonce, encodedEnvU, nil)
	// envU = append(envU, nonce...)
	return envU, nonce, pubUAsKey[:], salt, nil
}

func DeriveSessionToken(
	finData *oprf.FinalizeData,
	eval *oprf.Evaluation,
	envU,
	envUNonce,
	rwdUSalt,
	authNonce []byte,
) ([]byte, error) {
	oprfRet, err := finalizeRequest(finData, eval)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	rwdU, _, err := hardenOprfResult(oprfRet[0], rwdUSalt)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	privU, pubS, err := decryptEnvU(
		envU, envUNonce, rwdU)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	// Derive shared key
	var sharedKey x25519.Key
	ok := x25519.Shared(&sharedKey, privU, pubS)
	if !ok {
		return nil, errors.New("while deriving session key")
	}

	// Derive session token
	// TODO <22-04-2022, afjoseph> This can be faster if we init the slice with
	// x25519.Key + len(authNonce) instead of appending later
	b := make([]byte, x25519.Size)
	kdfr := hkdf.New(sha256.New, sharedKey[:], authNonce, nil)
	_, err = io.ReadFull(kdfr, b)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	st := []byte{}
	st = append(st, authNonce...)
	st = append(st, b...)
	return st, nil
}

// Construction of the encrypted envU is:
// ciphertext + tag + nonce
func decryptEnvU(
	envU,
	envUNonce,
	rwdU []byte,
) (*x25519.Key, *x25519.Key, error) {
	// Key derivation
	kdfr := hkdf.New(sha256.New, rwdU, nil, nil)
	aesKey := make([]byte, 16) // AES128-GCM
	_, err := io.ReadFull(kdfr, aesKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	// AES128-GCM decryption
	ciph, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}
	aesGcm, err := cipher.NewGCM(ciph)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}
	encodedEnvU, err := aesGcm.Open(nil, envUNonce, envU, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}
	var privU, pubS x25519.Key
	copy(privU[:], encodedEnvU[:x25519.Size])
	copy(pubS[:], encodedEnvU[x25519.Size:])
	return &privU, &pubS, nil
}
