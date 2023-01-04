package wallet

import (
	"encoding/hex"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/types"
	"testing"
)

func TestMsg(t *testing.T) {
	addr, _ := address.NewFromString("f1s6p5rqjg7msu6xoseyznniarazsyh5ukbned4yi")
	msg := &types.Message{
		To:    power.Address,
		From:  addr,
		Value: big.Zero(),

		Method: power.Methods.CreateMiner,
		Params: []byte{},

		GasLimit: 0,
	}
	var signMsg = types.SignedMessage{
		Message: *msg,
		Signature: crypto.Signature{
			Type: crypto.SigTypeSecp256k1,
			Data: []byte{1, 2, 3, 4},
		},
	}
	b, err := signMsg.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hex.EncodeToString(b))
}
