package wallet

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"golang.org/x/xerrors"
)

// todo get chain gas and nonce
func messageForSend(params SendParams) (*types.Message, error) {
	if params.From == address.Undef {
		return nil, xerrors.New("formAddr under")
	}

	msg := types.Message{
		From:  params.From,
		To:    params.To,
		Value: params.Val,

		Method: params.Method,
		Params: params.Params,
	}

	if params.GasPremium != nil {
		msg.GasPremium = *params.GasPremium
	} else {
		msg.GasPremium = types.NewInt(0)
	}

	if params.GasFeeCap != nil {
		msg.GasFeeCap = *params.GasFeeCap
	} else {
		msg.GasFeeCap = types.NewInt(0)
	}

	if params.GasLimit != nil {
		msg.GasLimit = *params.GasLimit
	} else {
		msg.GasLimit = 0
	}

	if params.Nonce != nil {
		msg.Nonce = *params.Nonce
	}

	return &msg, nil
}

func signMessage(account *wallet.Key, msg *types.Message) (*types.SignedMessage, error) {
	mb, err := msg.ToStorageBlock()
	if err != nil {
		return nil, xerrors.Errorf("serializing message: %w", err)
	}

	sig, err := sigs.Sign(wallet.ActSigType(account.Type), account.PrivateKey, mb.Cid().Bytes())
	if err != nil {
		return nil, xerrors.Errorf("failed to sign message: %w", err)
	}

	return &types.SignedMessage{
		Message:   *msg,
		Signature: *sig,
	}, nil
}
