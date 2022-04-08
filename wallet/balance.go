package wallet

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
)

func getBalance(addr string) (types.BigInt, error) {
	conf := config.Conf()
	result, err := client.NewClient(conf.Chain.RpcAddr, client.WalletBalance, []string{addr}).Call()
	if err != nil {
		return types.NewInt(0), err
	}

	r := client.Response{}
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
