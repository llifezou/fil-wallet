package wallet

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-cid"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"golang.org/x/xerrors"
)

func send(account *wallet.Key, message *types.Message) (cid.Cid, error) {
	var err error
	message, err = estimateMessageGasAndNonce(message)
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

	msgCid, err := client.LotusMpoolPush(conf.Chain.RpcAddr, conf.Chain.Token, signedMessage)
	if err != nil {
		return cid.Undef, err
	}

	return msgCid, nil
}
