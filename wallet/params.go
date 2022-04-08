package wallet

import (
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin"
	"github.com/urfave/cli/v2"
)

type SendParams struct {
	To   address.Address
	From address.Address
	Val  abi.TokenAmount

	GasPremium *abi.TokenAmount
	GasFeeCap  *abi.TokenAmount
	GasLimit   *int64

	Nonce  *uint64
	Method abi.MethodNum
	Params []byte
}

func getParams(cctx *cli.Context) (*SendParams, error) {
	var params SendParams
	var err error
	params.To, err = address.NewFromString(cctx.String("to"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target address: %w", err)
	}

	val, err := types.ParseFIL(cctx.String("amount"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}
	params.Val = abi.TokenAmount(val)

	if from := cctx.String("from"); from != "" {
		addr, err := address.NewFromString(from)
		if err != nil {
			return nil, err
		}

		params.From = addr
	}

	if cctx.IsSet("gas-premium") {
		gp, err := types.BigFromString(cctx.String("gas-premium"))
		if err != nil {
			return nil, err
		}
		params.GasPremium = &gp
	}

	if cctx.IsSet("gas-feecap") {
		gfc, err := types.BigFromString(cctx.String("gas-feecap"))
		if err != nil {
			return nil, err
		}
		params.GasFeeCap = &gfc
	}

	if cctx.IsSet("gas-limit") {
		limit := cctx.Int64("gas-limit")
		params.GasLimit = &limit
	}

	if cctx.IsSet("method") {
		params.Method = abi.MethodNum(cctx.Uint64("method"))
	} else { // transfer
		params.Method = builtin.MethodSend
	}

	if cctx.IsSet("params-json") {
		encParams, err := encodeParams(params.Method, cctx.String("params-json"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode json params: %w", err)
		}
		params.Params = encParams
	}
	if cctx.IsSet("params-hex") {
		if params.Params != nil {
			return nil, fmt.Errorf("can only specify one of 'params-json' and 'params-hex'")
		}
		decparams, err := hex.DecodeString(cctx.String("params-hex"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex params: %w", err)
		}
		params.Params = decparams
	}

	if cctx.IsSet("nonce") {
		n := cctx.Uint64("nonce")
		params.Nonce = &n
	}

	return &params, nil
}
