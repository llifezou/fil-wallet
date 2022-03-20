package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	logging "github.com/ipfs/go-log/v2"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"github.com/llifezou/fil-wallet/config"
	"github.com/llifezou/fil-wallet/util"
	"github.com/llifezou/hdwallet"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
)

var log = logging.Logger("wallet")
var WalletCmd = &cli.Command{
	Name:  "wallet",
	Usage: "fil wallet",
	Subcommands: []*cli.Command{
		mnemonicNew,
		walletNew,
		// todo walletSign,
		// todo walletVerify,
		// todo walletBalance,
		// todo walletTransfer
		// todo send msg
		// todo call fvm
	},
}

var mnemonicNew = &cli.Command{
	Name:  "mnemonic",
	Usage: "Generate a mnemonic",
	Action: func(_ *cli.Context) error {
		mnemonic, err := hdwallet.NewMnemonic()
		if err != nil {
			return err
		}

		color.Red("一定保存好助记词，丢失助记词将导致所有财产损失！")
		color.Red("Be sure to save mnemonic. Losing mnemonic will cause all property damage!")

		fmt.Printf("\n")
		color.Blue(mnemonic)
		fmt.Printf("\n")
		return nil
	},
}

var walletNew = &cli.Command{
	Name:  "generate",
	Usage: "Generate a key of the given type and index",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "type",
			Usage: "wallet type, ps: secp256k1, bls",
			Value: "secp256k1",
		},
		&cli.IntFlag{
			Name:  "index",
			Usage: "wallet index",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:  "export",
			Usage: "export key",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		t := cctx.String("type")

		var sigType crypto.SigType
		switch t {
		case "secp256k1":
			sigType = crypto.SigTypeSecp256k1
		case "bls":
			sigType = crypto.SigTypeBLS
		default:
			return xerrors.Errorf("--type: %s, TypeUnknown", t)
		}

		conf := config.Conf()
		if conf.Account.Mnemonic == "" {
			return xerrors.New("mnemonic is null")
		}

		if conf.Account.Password != "" {
			color.Red("钱包密码不为空，密码参与派生，请保存好密码！")
			color.Red("Wallet password is not empty, the password is involved in the derivation, please save the password!")

			fmt.Printf("\n")

			fmt.Println("请输入密码，确定 config.yaml 中的密码正确！")
			fmt.Println("Please enter the password to confirm that the password in the config.yaml is correct!")

			if password := util.GetPassword(); password != conf.Account.Password {
				color.Red(fmt.Sprintf("密码不匹配！输入的密码：%s", password))
				color.Red(fmt.Sprintf("Passwords do not match！The entered password: %s", password))
				os.Exit(1)
			}

			fmt.Printf("\n")
		}

		seed, err := hdwallet.GenerateSeedFromMnemonic(conf.Account.Mnemonic, conf.Account.Password)
		if err != nil {
			return err
		}

		index := cctx.Int("index")
		path := hdwallet.FilPath(index)
		log.Infow("wallet info", "type", t, "index", index, "path", path)

		extendSeed, err := hdwallet.GetExtendSeedFromPath(path, seed)
		if err != nil {
			return err
		}

		pk, err := sigs.Generate(sigType, extendSeed)
		if err != nil {
			return err
		}

		ki := types.KeyInfo{
			Type:       types.KeyType(t),
			PrivateKey: pk,
		}

		nk, err := wallet.NewKey(ki)
		if err != nil {
			return err
		}

		fmt.Println(nk.Address.String())

		if cctx.Bool("export") {
			b, err := json.Marshal(ki)
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			color.Blue(hex.EncodeToString(b))
		}

		return nil
	},
}
