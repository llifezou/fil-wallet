package wallet

import (
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-cid"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"golang.org/x/xerrors"
)

func send(account *wallet.Key, params SendParams) (cid.Cid, error) {
	message, err := buildMessage(params)
	if err != nil {
		return cid.Undef, err
	}

	if account.Address.String() != message.From.String() {
		return cid.Undef, xerrors.Errorf("The wallet address is: %s, from address is: %s", account.Address.String(), message.From.String())
	}

	conf := config.Conf()

	signedMessage, err := signMessage(account, message)
	if err != nil {
		return cid.Undef, err
	}

	msgCid, err := client.LotusMpoolPush(conf.Chain.RpcAddr, signedMessage)
	if err != nil {
		return cid.Undef, err
	}

	return msgCid, nil
}
