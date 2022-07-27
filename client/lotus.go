package client

import (
	"encoding/json"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func LotusWalletBalance(rpcAddr string, addr string) (types.BigInt, error) {
	result, err := NewClient(rpcAddr, WalletBalance, []string{addr}).Call()
	if err != nil {
		return types.NewInt(0), err
	}

	r := Response{}
	err = json.Unmarshal(result, &r)
	if err != nil {
		return types.NewInt(0), err
	}

	balance, err := types.BigFromString(r.Result.(string))
	if err != nil {
		return types.NewInt(0), err
	}

	return balance, nil
}

func LotusMpoolPush(rpcAddr string, signedMessage *types.SignedMessage) (cid.Cid, error) {
	result, err := NewClient(rpcAddr, MpoolPush, []types.SignedMessage{*signedMessage}).Call()
	if err != nil {
		return cid.Undef, err
	}

	r := Response{}
	err = json.Unmarshal(result, &r)
	if err != nil {
		return cid.Undef, err
	}
	if r.Error != nil {
		return cid.Undef, xerrors.Errorf("error: %s", r.Error.(map[string]interface{})["message"])
	}

	if r.Result != nil {
		return cid.Parse(r.Result.(map[string]interface{})["/"])
	}

	return cid.Undef, nil
}

func LotusGasEstimateMessageGas(rpcAddr string, message *types.Message, maxFee int64) (gasLimit float64, gasFeeCap, gasPremium string, err error) {
	var params []interface{}
	params = append(params, message)
	params = append(params, api.MessageSendSpec{MaxFee: abi.NewTokenAmount(maxFee)})
	params = append(params, types.EmptyTSK)

	result, err := NewClient(rpcAddr, GasEstimateMessageGas, params).Call()
	if err != nil {
		return 0, "", "", err
	}

	r := Response{}
	err = json.Unmarshal(result, &r)
	if err != nil {
		return 0, "", "", err
	}
	if r.Error != nil {
		return 0, "", "", xerrors.Errorf("error: %s", r.Error.(map[string]interface{})["message"])
	}

	if r.Result != nil {
		msgMap := r.Result.(map[string]interface{})
		gasLimit = msgMap["GasLimit"].(float64)
		gasFeeCap = msgMap["GasFeeCap"].(string)
		gasPremium = msgMap["GasPremium"].(string)
		return gasLimit, gasFeeCap, gasPremium, nil
	}

	return 0, "", "", xerrors.New("result is empty")
}

func LotusMpoolGetNonce(rpcAddr string, addr string) (float64, error) {
	result, err := NewClient(rpcAddr, MpoolGetNonce, []string{addr}).Call()
	if err != nil {
		return 0, err
	}

	r := Response{}
	err = json.Unmarshal(result, &r)
	if err != nil {
		return 0, err
	}

	return r.Result.(float64), nil
}
