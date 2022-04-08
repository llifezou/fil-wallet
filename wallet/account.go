package wallet

import (
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"github.com/llifezou/fil-wallet/config"
	"github.com/llifezou/hdwallet"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

func getAccount(cctx *cli.Context) (*wallet.Key, error) {
	t := cctx.String("type")

	var sigType crypto.SigType
	switch t {
	case "secp256k1":
		sigType = crypto.SigTypeSecp256k1
	case "bls":
		sigType = crypto.SigTypeBLS
	default:
		return nil, xerrors.Errorf("--type: %s, TypeUnknown", t)
	}

	conf := config.Conf()
	if conf.Account.Mnemonic == "" {
		return nil, xerrors.New("mnemonic is null")
	}

	seed, err := hdwallet.GenerateSeedFromMnemonic(conf.Account.Mnemonic, conf.Account.Password)
	if err != nil {
		return nil, err
	}

	index := cctx.Int("index")
	path := hdwallet.FilPath(index)
	log.Infow("wallet info", "type", t, "index", index, "path", path)

	extendSeed, err := hdwallet.GetExtendSeedFromPath(path, seed)
	if err != nil {
		return nil, err
	}

	pk, err := sigs.Generate(sigType, extendSeed)
	if err != nil {
		return nil, err
	}

	ki := types.KeyInfo{
		Type:       types.KeyType(t),
		PrivateKey: pk,
	}

	nk, err := wallet.NewKey(ki)
	if err != nil {
		return nil, err
	}

	return nk, nil
}
