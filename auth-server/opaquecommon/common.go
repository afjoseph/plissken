// This is a very boring file that has a **lot** of JSON (un)marshalling code
// for different objects so they can be handled cleanly by different parts of
// the code (running on different languages/platforms).
package opaquecommon

import (
	"encoding/hex"
	"encoding/json"

	"github.com/cloudflare/circl/group"
	"github.com/cloudflare/circl/oprf"
	"github.com/pkg/errors"
)

var OprfSuiteID = oprf.SuiteP256
var G = group.P256

type OprfRequestResults struct {
	*innerOprfRequestResults
	Username string                  `json:"-"`
	AppToken string                  `json:"-"`
	Inputs   [][]byte                `json:"-"`
	FinData  *oprf.FinalizeData      `json:"-"`
	EvalReq  *oprf.EvaluationRequest `json:"-"`
}

type innerOprfRequestResults struct {
	Username                  string   `json:"username"`
	AppToken                  string   `json:"apptoken"`
	HexEncodedInputs          []string `json:"inputs"`
	HexEncodedBlinds          []string `json:"blinds"`
	HexEncodedEvalReqElements []string `json:"eval_req_elements"`
}

func (d *OprfRequestResults) MarshalJSON() ([]byte, error) {
	blinds := []string{}
	for _, bl := range d.FinData.CopyBlinds() {
		sbl, err := bl.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		blinds = append(blinds, hex.EncodeToString(sbl))
	}

	inputs := []string{}
	for _, bl := range d.Inputs {
		inputs = append(inputs, hex.EncodeToString(bl))
	}

	evalReqElements := []string{}
	for _, bl := range d.EvalReq.Elements {
		sbl, err := bl.MarshalBinaryCompress()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		evalReqElements = append(evalReqElements, hex.EncodeToString(sbl))
	}

	return json.Marshal(&innerOprfRequestResults{
		Username:                  d.Username,
		AppToken:                  d.AppToken,
		HexEncodedInputs:          inputs,
		HexEncodedBlinds:          blinds,
		HexEncodedEvalReqElements: evalReqElements,
	})
}

func (d *OprfRequestResults) UnmarshalJSON(data []byte) error {
	di := &innerOprfRequestResults{}
	err := json.Unmarshal(data, di)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.innerOprfRequestResults = di
	d.Username = di.Username
	d.AppToken = di.AppToken

	d.Inputs = [][]byte{}
	for _, sin := range d.HexEncodedInputs {
		in, err := hex.DecodeString(sin)
		if err != nil {
			return errors.Wrap(err, "")
		}
		d.Inputs = append(d.Inputs, in)
	}

	blinds := []oprf.Blind{}
	for _, sbl := range d.HexEncodedBlinds {
		blAsByteSlice, err := hex.DecodeString(sbl)
		if err != nil {
			return errors.Wrap(err, "")
		}
		bl := G.NewScalar()
		err = bl.UnmarshalBinary(blAsByteSlice)
		if err != nil {
			return errors.Wrap(err, "")
		}
		blinds = append(blinds, bl)
	}

	d.FinData, d.EvalReq, err = oprf.NewClient(OprfSuiteID).
		DeterministicBlind(d.Inputs, blinds)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

type OprfServerEvaluation struct {
	*innerOprfServerEvaluation
	Eval *oprf.Evaluation `json:"-"`
}

type innerOprfServerEvaluation struct {
	HexEncodedElements []string `json:"elements"`
}

func (d *OprfServerEvaluation) MarshalJSON() ([]byte, error) {
	arr := []string{}
	for _, el := range d.Eval.Elements {
		sel, err := el.MarshalBinaryCompress()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		arr = append(arr, hex.EncodeToString(sel))
	}

	return json.Marshal(&innerOprfServerEvaluation{
		HexEncodedElements: arr,
	})
}

func (d *OprfServerEvaluation) UnmarshalJSON(data []byte) error {
	di := &innerOprfServerEvaluation{}
	err := json.Unmarshal(data, di)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.innerOprfServerEvaluation = di

	d.Eval = &oprf.Evaluation{}
	d.Eval.Elements = []oprf.Evaluated{}
	for _, hsel := range d.HexEncodedElements {
		sel, err := hex.DecodeString(hsel)
		if err != nil {
			return errors.Wrap(err, "")
		}
		el := G.NewElement()
		err = el.UnmarshalBinary(sel)
		if err != nil {
			return errors.Wrap(err, "")
		}
		d.Eval.Elements = append(d.Eval.Elements, el)
	}
	return nil
}

type PasswordRegistrationData struct {
	*innerPasswordRegistrationData
	Username  string `json:"-"`
	AppToken  string `json:"-"`
	EnvU      []byte `json:"-"`
	EnvUNonce []byte `json:"-"`
	PubU      []byte `json:"-"`
	Salt      []byte `json:"-"`
}

type innerPasswordRegistrationData struct {
	Username            string `json:"username"`
	AppToken            string `json:"apptoken"`
	HexEncodedEnvU      string `json:"envu"`
	HexEncodedEnvUNonce string `json:"envu_nonce"`
	HexEncodedPubU      string `json:"pubu"`
	HexEncodedSalt      string `json:"salt"`
}

func (d *PasswordRegistrationData) MarshalJSON() ([]byte, error) {
	return json.Marshal(&innerPasswordRegistrationData{
		Username:            d.Username,
		AppToken:            d.AppToken,
		HexEncodedEnvU:      hex.EncodeToString(d.EnvU),
		HexEncodedEnvUNonce: hex.EncodeToString(d.EnvUNonce),
		HexEncodedPubU:      hex.EncodeToString(d.PubU),
		HexEncodedSalt:      hex.EncodeToString(d.Salt),
	})
}

func (d *PasswordRegistrationData) UnmarshalJSON(data []byte) error {
	di := &innerPasswordRegistrationData{}
	err := json.Unmarshal(data, di)
	if err != nil {
		return err
	}
	d.innerPasswordRegistrationData = di

	d.Username = d.innerPasswordRegistrationData.Username
	d.AppToken = d.innerPasswordRegistrationData.AppToken
	d.EnvU, err = hex.DecodeString(d.HexEncodedEnvU)
	if err != nil {
		return err
	}
	d.EnvUNonce, err = hex.DecodeString(d.HexEncodedEnvUNonce)
	if err != nil {
		return err
	}
	d.PubU, err = hex.DecodeString(d.HexEncodedPubU)
	if err != nil {
		return err
	}
	d.Salt, err = hex.DecodeString(d.HexEncodedSalt)
	if err != nil {
		return err
	}
	return nil
}

// type SerializedEncryptedUserData struct {
// 	*innerSerializedEncryptedUserData
// 	EnvU      []byte `json:"-"`
// 	EnvUNonce []byte `json:"-"`
// }

// type innerSerializedEncryptedUserData struct {
// 	HexEncodedEnvU      string `json:"envu"`
// 	HexEncodedEnvUNonce string `json:"envu_nonce"`
// }

// func (d *SerializedEncryptedUserData) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(&innerSerializedEncryptedUserData{
// 		HexEncodedEnvU:      hex.EncodeToString(d.EnvU),
// 		HexEncodedEnvUNonce: hex.EncodeToString(d.EnvUNonce),
// 	})
// }

// func (d *SerializedEncryptedUserData) UnmarshalJSON(data []byte) error {
// 	di := &innerSerializedEncryptedUserData{}
// 	err := json.Unmarshal(data, di)
// 	if err != nil {
// 		return err
// 	}
// 	d.innerSerializedEncryptedUserData = di
// 	d.EnvU, err = hex.DecodeString(d.HexEncodedEnvU)
// 	if err != nil {
// 		return err
// 	}
// 	d.EnvUNonce, err = hex.DecodeString(d.HexEncodedEnvU)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// type SerializedSessionKey struct {
// 	*innerSerializedSessionKey
// 	SessionKey []byte `json:"-"`
// }

// type innerSerializedSessionKey struct {
// 	HexEncodedSessionKey string `json:"session_key"`
// }

// func (d *SerializedSessionKey) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(&innerSerializedSessionKey{
// 		HexEncodedSessionKey: hex.EncodeToString(d.SessionKey),
// 	})
// }

// func (d *SerializedSessionKey) UnmarshalJSON(data []byte) error {
// 	di := &innerSerializedSessionKey{}
// 	err := json.Unmarshal(data, di)
// 	if err != nil {
// 		return err
// 	}
// 	d.innerSerializedSessionKey = di
// 	d.SessionKey, err = hex.DecodeString(d.HexEncodedSessionKey)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

type StartPasswordAuthServerResp struct {
	*innerStartPasswordAuthServerResp
	Eval      *oprf.Evaluation `json:"-"`
	EnvU      []byte           `json:"-"`
	EnvUNonce []byte           `json:"-"`
	RwdUSalt  []byte           `json:"-"`
	AuthNonce []byte           `json:"-"`
}

type innerStartPasswordAuthServerResp struct {
	HexEncodedElements  []string `json:"elements"`
	HexEncodedEnvU      string   `json:"envu"`
	HexEncodedEnvUNonce string   `json:"envu_nonce"`
	HexEncodedRwdUSalt  string   `json:"rwdu_salt"`
	HexEncodedAuthNonce string   `json:"auth_nonce"`
}

func (d *StartPasswordAuthServerResp) MarshalJSON() ([]byte, error) {
	elements := []string{}
	for _, el := range d.Eval.Elements {
		sel, err := el.MarshalBinaryCompress()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		elements = append(elements, hex.EncodeToString(sel))
	}

	return json.Marshal(&innerStartPasswordAuthServerResp{
		HexEncodedElements:  elements,
		HexEncodedEnvU:      hex.EncodeToString(d.EnvU),
		HexEncodedEnvUNonce: hex.EncodeToString(d.EnvUNonce),
		HexEncodedRwdUSalt:  hex.EncodeToString(d.RwdUSalt),
		HexEncodedAuthNonce: hex.EncodeToString(d.AuthNonce),
	})
}

func (d *StartPasswordAuthServerResp) UnmarshalJSON(data []byte) error {
	di := &innerStartPasswordAuthServerResp{}
	err := json.Unmarshal(data, di)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.innerStartPasswordAuthServerResp = di
	d.Eval = &oprf.Evaluation{}
	d.Eval.Elements = []oprf.Evaluated{}
	for _, hsel := range d.HexEncodedElements {
		sel, err := hex.DecodeString(hsel)
		if err != nil {
			return errors.Wrap(err, "")
		}
		el := G.NewElement()
		err = el.UnmarshalBinary(sel)
		if err != nil {
			return errors.Wrap(err, "")
		}
		d.Eval.Elements = append(d.Eval.Elements, el)
	}
	d.EnvU, err = hex.DecodeString(d.HexEncodedEnvU)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.EnvUNonce, err = hex.DecodeString(d.HexEncodedEnvUNonce)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.RwdUSalt, err = hex.DecodeString(d.HexEncodedRwdUSalt)
	if err != nil {
		return errors.Wrap(err, "")
	}
	d.AuthNonce, err = hex.DecodeString(d.HexEncodedAuthNonce)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

type FinalizePasswordAuthData struct {
	Username     string `json:"username"`
	AppToken     string `json:"apptoken"`
	SessionToken string `json:"session_token"`
}
