package wallet

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-cid"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

func send(cctx *cli.Context, account *wallet.Key, params SendParams) (cid.Cid, error) {
	message, err := messageForSend(params)
	if err != nil {
		return cid.Undef, err
	}

	if account.Address.String() != message.From.String() {
		return cid.Undef, xerrors.Errorf("type: %s, index:%d, The derived address is: %s, from address is: %s", cctx.String("type"), cctx.Int("index"), account.Address.String(), message.From.String())
	}

	signedMessage, err := signMessage(account, message)
	if err != nil {
		return cid.Undef, err
	}

	conf := config.Conf()
	result, err := client.NewClient(conf.Chain.RpcAddr, client.MpoolPush, []types.SignedMessage{*signedMessage}).Call()
	if err != nil {
		return cid.Undef, err
	}

	r := client.Response{}
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
