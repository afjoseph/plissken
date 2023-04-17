package common

import (
	"encoding/json"
	"testing"

	"github.com/cloudflare/circl/oprf"
	"github.com/stretchr/testify/require"
)

func TestSerializations(t *testing.T) {
	t.Run("Serializae OprfRequestResults", func(t *testing.T) {
		inputs := [][]byte{[]byte("bunnyfoofoo")}
		finData, evalReq, err := oprf.NewClient(OprfSuiteID).Blind(inputs)
		require.NoError(t, err)
		ret := &OprfRequestResults{
			Inputs:  inputs,
			FinData: finData,
			EvalReq: evalReq}
		b, err := json.Marshal(ret)
		require.NoError(t, err)

		var ret2 OprfRequestResults
		err = json.Unmarshal(b, &ret2)
		require.NoError(t, err)

		require.Equal(t, ret.Inputs, ret2.Inputs)
		require.Equal(t, ret.FinData, ret2.FinData)
		require.Equal(t, ret.EvalReq, ret2.EvalReq)
	})
}
