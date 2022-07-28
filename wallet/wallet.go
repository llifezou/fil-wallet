package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin"
	logging "github.com/ipfs/go-log/v2"
	"github.com/llifezou/fil-sdk/sigs"
	_ "github.com/llifezou/fil-sdk/sigs/bls"
	_ "github.com/llifezou/fil-sdk/sigs/secp"
	"github.com/llifezou/fil-wallet/config"
	"github.com/llifezou/hdwallet"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("wallet")
var Cmd = &cli.Command{
	Name:  "wallet",
	Usage: "fil wallet",
	Subcommands: []*cli.Command{
		mnemonicNew,
		walletNew,
		walletSign,
		walletVerify,
		walletBalance,
		walletTransfer,
		walletSendCmd,
		multisigCmd,
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
		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		fmt.Println(nk.Address.String())

		if cctx.Bool("export") {
			b, err := json.Marshal(nk.KeyInfo)
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			color.Blue(hex.EncodeToString(b))
		}

		return nil
	},
}

var walletSign = &cli.Command{
	Name:  "sign",
	Usage: "Sign a message",
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
	},
	ArgsUsage: "<signing address> <hexMessage>",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() || cctx.NArg() != 2 {
			return fmt.Errorf("must specify signing address and message to sign")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}
		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		if nk.Address.String() != addr.String() {
			return xerrors.Errorf("The wallet address is: %s, sign address is: %s", nk.Address.String(), addr.String())
		}

		msg, err := hex.DecodeString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		sig, err := sigs.Sign(wallet.ActSigType(nk.Type), nk.PrivateKey, msg)
		if err != nil {
			return err
		}

		sigBytes := append([]byte{byte(sig.Type)}, sig.Data...)

		fmt.Println(hex.EncodeToString(sigBytes))
		return nil
	},
}

var walletVerify = &cli.Command{
	Name:  "verify",
	Usage: "Verify the signature of a message",
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
	},
	ArgsUsage: "<signing address> <hexMessage> <signature>",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() || cctx.NArg() != 3 {
			return fmt.Errorf("must specify signing address, message, and signature to verify")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		if nk.Address.String() != addr.String() {
			return xerrors.Errorf("The wallet address is: %s, sign address is: %s", nk.Address.String(), addr.String())
		}

		msg, err := hex.DecodeString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		sigBytes, err := hex.DecodeString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		var sig crypto.Signature
		if err := sig.UnmarshalBinary(sigBytes); err != nil {
			return err
		}

		err = sigs.Verify(&sig, addr, msg)
		if err != nil {
			fmt.Println("invalid signature")
			return err
		}
		fmt.Println("valid signature")
		return nil
	},
}

var walletBalance = &cli.Command{
	Name:      "balance",
	Usage:     "Get account balance",
	ArgsUsage: "[address]",
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
	},
	Action: func(cctx *cli.Context) error {
		var addr address.Address
		var err error
		if cctx.Args().First() != "" {
			addr, err = address.NewFromString(cctx.Args().First())
		} else {
			key, err := getAccount(cctx)
			if err != nil {
				return err
			}
			addr = key.Address
		}
		if err != nil {
			return err
		}

		balance, err := getBalance(addr.String())
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if balance.Equals(types.NewInt(0)) {
			fmt.Printf("%s (warning: may display 0 if chain sync in progress)\n", types.FIL(balance))
		} else {
			fmt.Printf("%s\n", types.FIL(balance))
		}

		return nil
	},
}

var walletTransfer = &cli.Command{
	Name:  "transfer",
	Usage: "Transfer funds between accounts",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "optionally specify the account to send funds to",
		},
		&cli.StringFlag{
			Name:  "amount",
			Usage: "transfer amount",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "specify gas price to use in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "specify gas fee cap to use in AttoFIL",
			Value: "0",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "specify gas limit",
			Value: 0,
		},
		&cli.Uint64Flag{
			Name:  "nonce",
			Usage: "specify the nonce to use",
			Value: 0,
		},
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
	},
	Action: func(cctx *cli.Context) error {
		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		sendParams, err := getParams(cctx)
		if err != nil {
			return err
		}

		sendMessage, err := buildMessage(sendParams)
		if err != nil {
			return err
		}

		messageCid, err := send(nk, sendMessage)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, messageCid.String()))
		return nil
	},
}

var walletSendCmd = &cli.Command{
	Name:  "send",
	Usage: "Send funds between accounts",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "optionally specify the account to send funds to",
		},
		&cli.StringFlag{
			Name:  "amount",
			Usage: "transfer amount",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "specify gas price to use in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "specify gas fee cap to use in AttoFIL",
			Value: "0",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "specify gas limit",
			Value: 0,
		},
		&cli.Uint64Flag{
			Name:  "nonce",
			Usage: "specify the nonce to use",
			Value: 0,
		},
		&cli.Uint64Flag{
			Name:  "method",
			Usage: "specify method to invoke",
			Value: uint64(builtin.MethodSend),
		},
		&cli.StringFlag{
			Name:  "params-json",
			Usage: "specify invocation parameters in json",
		},
		&cli.StringFlag{
			Name:  "params-hex",
			Usage: "specify invocation parameters in hex",
		},
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
	},
	Action: func(cctx *cli.Context) error {
		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		sendParams, err := getParams(cctx)
		if err != nil {
			return err
		}

		sendMessage, err := buildMessage(sendParams)
		if err != nil {
			return err
		}

		messageCid, err := send(nk, sendMessage)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, messageCid.String()))

		return nil
	},
}
