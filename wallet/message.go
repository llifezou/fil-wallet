package wallet

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"golang.org/x/xerrors"
)

func buildMessage(params SendParams) (*types.Message, error) {
	if params.From == address.Undef {
		return nil, xerrors.New("formAddr undef")
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

	if msg.GasLimit == 0 ||
		msg.GasFeeCap == types.EmptyInt || types.BigCmp(msg.GasFeeCap, types.NewInt(0)) == 0 ||
		msg.GasPremium == types.EmptyInt || types.BigCmp(msg.GasPremium, types.NewInt(0)) == 0 {

		conf := config.Conf()
		gasLimit, gasFeeCap, gasPremium, err := client.LotusGasEstimateMessageGas(conf.Chain.RpcAddr, &msg, types.MustParseFIL(conf.Chain.MaxFee).Int64())
		if err != nil {
			return nil, err
		}

		if msg.GasLimit == 0 {
			msg.GasLimit = int64(gasLimit)
		}

		if msg.GasFeeCap == types.EmptyInt || types.BigCmp(msg.GasFeeCap, types.NewInt(0)) == 0 {
			gasFeeCapBigInt, err := types.BigFromString(gasFeeCap)
			if err != nil {
				return nil, err
			}
			msg.GasFeeCap = gasFeeCapBigInt
		}

		if msg.GasPremium == types.EmptyInt || types.BigCmp(msg.GasPremium, types.NewInt(0)) == 0 {
			gasPremiumBigInt, err := types.BigFromString(gasPremium)
			if err != nil {
				return nil, err
			}
			msg.GasPremium = gasPremiumBigInt
		}
	}

	if params.Nonce != nil {
		msg.Nonce = *params.Nonce
	} else {
		nonce, err := client.LotusMpoolGetNonce(config.Conf().Chain.RpcAddr, msg.From.String())
		if err != nil {
			return nil, err
		}

		msg.Nonce = uint64(nonce)
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
