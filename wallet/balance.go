package wallet

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
)

func getBalance(addr string) (types.BigInt, error) {
	conf := config.Conf()

	balance, err := client.LotusWalletBalance(conf.Chain.RpcAddr, addr)
	if err != nil {
		return types.NewInt(0), err
	}

	return balance, nil
}
