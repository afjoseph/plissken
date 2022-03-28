package main

import (
	"encoding/hex"
	"encoding/json"

	"github.com/afjoseph/plissken-auth-server/opaqueclient"
	"github.com/afjoseph/plissken-auth-server/opaquecommon"
	"github.com/cloudflare/circl/dh/x25519"
	"github.com/gopherjs/gopherjs/js"
	"github.com/pkg/errors"
)

func makeOprfRequest(
	apptoken, username, password string,
	// Returns a JSON-Marshalled OprfRequestResults
) string {
	if username == "" || password == "" {
		panic("Username or password are empty")
	}

	inputs, finData, evalReq, err := opaqueclient.MakeOprfRequest(password)
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}
	b, err := json.Marshal(
		&opaquecommon.OprfRequestResults{
			Username: username,
			AppToken: apptoken,
			Inputs:   inputs,
			FinData:  finData,
			EvalReq:  evalReq})
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}
	return string(b)
}

func finalizePasswordRegistration(
	apptoken,
	username,
	oprfReqJsonStr,
	oprfServerEvalJsonStr,
	hexEncodedServerPubKey string,
	// Returns a JSON-Marshalled PasswordRegistrationData
) string {
	println("Finalizing password reg request with %v, %v and %v",
		username, oprfReqJsonStr, oprfServerEvalJsonStr)

	// Decode things
	// - server pub key
	b, err := hex.DecodeString(hexEncodedServerPubKey)
	if err != nil {
		panic(errors.Wrap(err, "decoding hex key").Error())
	}
	var serverPubKey x25519.Key
	copy(serverPubKey[:], b)
	// - oprf request
	oprfReq := &opaquecommon.OprfRequestResults{}
	err = json.Unmarshal([]byte(oprfReqJsonStr), oprfReq)
	if err != nil {
		panic(errors.Wrap(err, "while decoding oprf request").Error())
	}
	// - oprf server evaluation
	oprfServerEval := &opaquecommon.OprfServerEvaluation{}
	err = json.Unmarshal([]byte(oprfServerEvalJsonStr), oprfServerEval)
	if err != nil {
		panic(errors.Wrap(err, "while decoding server eval").Error())
	}

	// Make EnvU
	envU, envUNonce,
		pubU, salt, err := opaqueclient.MakeEnvU(
		oprfReq.FinData,
		oprfServerEval.Eval,
		serverPubKey)
	if err != nil {
		panic(errors.Wrap(err, "while making envu").Error())
	}

	// Serialize and return
	b, err = json.Marshal(&opaquecommon.PasswordRegistrationData{
		AppToken:  apptoken,
		Username:  username,
		EnvU:      envU,
		EnvUNonce: envUNonce,
		PubU:      pubU,
		Salt:      salt,
	})
	if err != nil {
		panic(errors.Wrap(err, "while making password reg data").Error())
	}
	return string(b)
}

func finalizePasswordAuthentication(
	username,
	oprfReqJsonStr,
	startPasswordAuthDataJsonStr,
	hexEncodedServerPubKey string,
) string {
	println("Finalizing password-auth request")

	// Decode server pub key
	b, err := hex.DecodeString(hexEncodedServerPubKey)
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}
	var serverPubKey x25519.Key
	copy(serverPubKey[:], b)

	// Decode oprf request
	oprfReq := &opaquecommon.OprfRequestResults{}
	err = json.Unmarshal([]byte(oprfReqJsonStr), oprfReq)
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}

	// Decode Password authentication server response
	startPasswordAuthData := &opaquecommon.StartPasswordAuthServerResp{}
	err = json.Unmarshal(
		[]byte(startPasswordAuthDataJsonStr), startPasswordAuthData)
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}

	sessionToken, err := opaqueclient.DeriveSessionToken(
		oprfReq.FinData,
		startPasswordAuthData.Eval,
		startPasswordAuthData.EnvU, startPasswordAuthData.EnvUNonce,
		startPasswordAuthData.RwdUSalt,
		startPasswordAuthData.AuthNonce,
	)
	if err != nil {
		panic(errors.Wrap(err, "").Error())
	}
	return hex.EncodeToString(sessionToken)
}

func main() {
	js.Module.Get("exports").Set("make_oprf_request", makeOprfRequest)
	js.Module.Get("exports").Set("finalize_password_registration",
		finalizePasswordRegistration)
	js.Module.Get("exports").Set("finalize_password_authentication",
		finalizePasswordAuthentication)
}
