package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet/key"
	_ "github.com/filecoin-project/lotus/lib/sigs/bls"
	_ "github.com/filecoin-project/lotus/lib/sigs/secp"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"github.com/llifezou/fil-wallet/config"
	"github.com/llifezou/fil-wallet/util"
	"github.com/llifezou/hdwallet"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
	"strings"
)

func getAccount(cctx *cli.Context) (*key.Key, error) {
	conf := config.Conf()

	if conf.Account.Key != "" {
		log.Info("key is not empty, key will be used instead of mnemonic")
		var ki types.KeyInfo
		switch conf.Account.KeyFormat {
		case "hex-lotus":
			data, err := hex.DecodeString(strings.TrimSpace(string(conf.Account.Key)))
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(data, &ki); err != nil {
				return nil, err
			}
		case "json-lotus":
			if err := json.Unmarshal([]byte(conf.Account.Key), &ki); err != nil {
				return nil, err
			}
		case "gfc-json":
			var f struct {
				KeyInfo []struct {
					PrivateKey []byte
					SigType    int
				}
			}
			if err := json.Unmarshal([]byte(conf.Account.Key), &f); err != nil {
				return nil, xerrors.Errorf("failed to parse go-filecoin key: %s", err)
			}

			gk := f.KeyInfo[0]
			ki.PrivateKey = gk.PrivateKey
			switch gk.SigType {
			case 1:
				ki.Type = types.KTSecp256k1
			case 2:
				ki.Type = types.KTBLS
			default:
				return nil, fmt.Errorf("unrecognized key type: %d", gk.SigType)
			}
		default:
			return nil, fmt.Errorf("unrecognized format: %s", cctx.String("format"))
		}

		nk, err := key.NewKey(ki)
		if err != nil {
			return nil, err
		}

		return nk, nil
	}

	if conf.Account.Mnemonic == "" {
		return nil, xerrors.New("mnemonic is null")
	}

	var password = ""
	if conf.Account.Password {
		color.Red("密码参与派生，请保存好密码！")
		color.Red("the password is involved in the derivation, please save the password!")

		fmt.Printf("\n")

		var err error
		password, err = util.GetPassword(true)
		if err != nil {
			color.Red(fmt.Sprintf("Failed get password: %s", err.Error()))
			os.Exit(1)
		}
	}

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

	seed, err := hdwallet.GenerateSeedFromMnemonic(conf.Account.Mnemonic, password)
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

	nk, err := key.NewKey(ki)
	if err != nil {
		return nil, err
	}

	return nk, nil
}

func getAccountList(cctx *cli.Context, index int) (*key.Key, error) {
	conf := config.Conf()

	if conf.Account.Mnemonic == "" {
		return nil, xerrors.New("mnemonic is null")
	}

	var password = ""
	if conf.Account.Password {
		color.Red("密码参与派生，请保存好密码！")
		color.Red("the password is involved in the derivation, please save the password!")

		fmt.Printf("\n")

		var err error
		password, err = util.GetPassword(true)
		if err != nil {
			color.Red(fmt.Sprintf("Failed get password: %s", err.Error()))
			os.Exit(1)
		}
	}

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

	seed, err := hdwallet.GenerateSeedFromMnemonic(conf.Account.Mnemonic, password)
	if err != nil {
		return nil, err
	}

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

	nk, err := key.NewKey(ki)
	if err != nil {
		return nil, err
	}

	return nk, nil
}
