package wallet

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors"
	bt2 "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/types"
	init2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	msig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"strconv"
	"time"
)

var multisigCmd = &cli.Command{
	Name:  "msig",
	Usage: "Interact with a multisig wallet",
	Flags: []cli.Flag{
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
	Subcommands: []*cli.Command{
		msigCreateCmd,
		msigProposeCmd,
		msigRemoveProposeCmd,
		msigApproveCmd,
		msigCancelCmd,
		msigTransferProposeCmd,
		msigTransferApproveCmd,
		msigAddProposeCmd,
		msigAddApproveCmd,
		msigAddCancelCmd,
		msigSwapProposeCmd,
		msigSwapApproveCmd,
		msigSwapCancelCmd,
		msigLockProposeCmd,
		msigLockApproveCmd,
		msigLockCancelCmd,
		msigThresholdProposeCmd,
		msigThresholdApproveCmd,
		msigChangeOwnerProposeCmd,
		msigChangeOwnerApproveCmd,
		msigWithdrawBalanceProposeCmd,
		msigWithdrawBalanceApproveCmd,
	},
}

var msigCreateCmd = &cli.Command{
	Name:      "create",
	Usage:     "Create a new multisig wallet",
	ArgsUsage: "[address1 address2 ...]",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "required",
			Usage: "number of required approvals (uses number of signers provided if omitted)",
		},
		&cli.StringFlag{
			Name:  "value",
			Usage: "initial funds to give to multisig",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "duration",
			Usage: "length of the period over which funds unlock",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the create message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 1 {
			return fmt.Errorf("multisigs must have at least one signer")
		}

		var addrs []address.Address
		for _, a := range cctx.Args().Slice() {
			addr, err := address.NewFromString(a)
			if err != nil {
				return err
			}
			addrs = append(addrs, addr)
		}

		var sendAddr address.Address
		addr, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}

		sendAddr = addr

		val := cctx.String("value")
		filval, err := types.ParseFIL(val)
		if err != nil {
			return err
		}

		intVal := types.BigInt(filval)

		required := cctx.Uint64("required")
		if required == 0 {
			required = uint64(len(addrs))
		}

		d := abi.ChainEpoch(cctx.Uint64("duration"))

		gp := types.NewInt(1)

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()
		proto, err := msiger.MsigCreate(required, addrs, d, intVal, sendAddr, gp)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent create in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		fmt.Println("message waiting for confirmation...")
		conf := config.Conf()
		var wait *client.MsgLookup
		var i int = 0
		for {
			if i > 60 {
				return xerrors.New("wait timeout")
			}

			time.Sleep(30 * time.Second)
			wait, err = client.LotusStateSearchMsg(conf.Chain.RpcAddr, conf.Chain.Token, msgCid.String())
			if err != nil {
				log.Error(err)
				return err
			}

			if wait == nil {
				i++
				continue
			}

			break
		}

		if wait.Receipt.ExitCode != 0 {
			return fmt.Errorf("msg returned exit %d", wait.Receipt.ExitCode)
		}

		var execreturn init2.ExecReturn
		if err := execreturn.UnmarshalCBOR(bytes.NewReader(wait.Receipt.Return)); err != nil {
			return err
		}
		fmt.Fprintln(cctx.App.Writer, "Created new multisig: ", execreturn.IDAddress, execreturn.RobustAddress)

		return nil
	},
}

var msigProposeCmd = &cli.Command{
	Name:      "propose",
	Usage:     "Propose a multisig transaction",
	ArgsUsage: "[multisigAddress destinationAddress value <methodId methodParams> (optional)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the propose message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 3 {
			return fmt.Errorf("must pass at least multisig address, destination, and value")
		}

		if cctx.Args().Len() > 3 && cctx.Args().Len() != 5 {
			return fmt.Errorf("must either pass three or five arguments")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		dest, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		value, err := types.ParseFIL(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		var method uint64
		var params []byte
		if cctx.Args().Len() == 5 {
			m, err := strconv.ParseUint(cctx.Args().Get(3), 10, 64)
			if err != nil {
				return err
			}
			method = m

			p, err := hex.DecodeString(cctx.Args().Get(4))
			if err != nil {
				return err
			}
			params = p
		}

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		conf := config.Conf()
		code, _, _, _, err := client.LotusStateGetActor(conf.Chain.RpcAddr, conf.Chain.Token, msig.String())
		if err != nil {
			return fmt.Errorf("failed to look up multisig %s: %w", msig, err)
		}

		codeCid, err := cid.Parse(code)
		if err != nil {
			return fmt.Errorf("failed to cid.Parse %s: %w", code, err)
		}

		if !bt2.IsMultisigActor(codeCid) {
			return fmt.Errorf("actor %s is not a multisig actor", msig)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()
		proto, err := msiger.MsigPropose(msig, dest, types.BigInt(value), from, method, params)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigApproveCmd = &cli.Command{
	Name:      "approve",
	Usage:     "Approve a multisig message",
	ArgsUsage: "<multisigAddress messageId> [proposerAddress destination value [methodId methodParams]]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 2 {
			return fmt.Errorf("must pass at least multisig address and message ID")
		}

		if cctx.Args().Len() > 2 && cctx.Args().Len() < 5 {
			return fmt.Errorf("usage: msig approve <msig addr> <message ID> <proposer address> <desination> <value>")
		}

		if cctx.Args().Len() > 5 && cctx.Args().Len() != 7 {
			return fmt.Errorf("usage: msig approve <msig addr> <message ID> <proposer address> <desination> <value> [ <method> <params> ]")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		var msgCid cid.Cid
		if cctx.Args().Len() == 2 {
			proto, err := msiger.MsigApprove(msig, txid, from)
			if err != nil {
				return err
			}

			msgCid, err = send(nk, proto)
			if err != nil {
				log.Error(err)
				return err
			}
		} else {
			proposer, err := address.NewFromString(cctx.Args().Get(2))
			if err != nil {
				return err
			}

			if proposer.Protocol() != address.ID {
				proposerID, err := client.LotusStateLookupID(config.Conf().Chain.RpcAddr, config.Conf().Chain.Token, proposer.String())
				if err != nil {
					return err
				}
				proposer, err = address.NewFromString(proposerID)
				if err != nil {
					return err
				}
			}

			dest, err := address.NewFromString(cctx.Args().Get(3))
			if err != nil {
				return err
			}

			value, err := types.ParseFIL(cctx.Args().Get(4))
			if err != nil {
				return err
			}

			var method uint64
			var params []byte
			if cctx.Args().Len() == 7 {
				m, err := strconv.ParseUint(cctx.Args().Get(5), 10, 64)
				if err != nil {
					return err
				}
				method = m

				p, err := hex.DecodeString(cctx.Args().Get(6))
				if err != nil {
					return err
				}
				params = p
			}

			proto, err := msiger.MsigApproveTxnHash(msig, txid, proposer, dest, types.BigInt(value), from, method, params)
			if err != nil {
				return err
			}

			msgCid, err = send(nk, proto)
			if err != nil {
				log.Error(err)
				return err
			}
		}

		fmt.Println("sent approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigTransferProposeCmd = &cli.Command{
	Name:      "transfer-propose",
	Usage:     "Propose a multisig transaction",
	ArgsUsage: "[multisigAddress destinationAddress value",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the propose message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 3 {
			return fmt.Errorf("must have multisig address, destination, and value")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		dest, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		value, err := types.ParseFIL(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		var method uint64
		var params []byte

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		conf := config.Conf()
		code, _, _, _, err := client.LotusStateGetActor(conf.Chain.RpcAddr, conf.Chain.Token, msig.String())
		if err != nil {
			return fmt.Errorf("failed to look up multisig %s: %w", msig, err)
		}

		codeCid, err := cid.Parse(code)
		if err != nil {
			return fmt.Errorf("failed to cid.Parse %s: %w", code, err)
		}

		if !bt2.IsMultisigActor(codeCid) {
			return fmt.Errorf("actor %s is not a multisig actor", msig)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()
		proto, err := msiger.MsigPropose(msig, dest, types.BigInt(value), from, method, params)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("transfer proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigTransferApproveCmd = &cli.Command{
	Name:      "transfer-approve",
	Usage:     "Approve a multisig message",
	ArgsUsage: "<multisigAddress messageId>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("must have multisig address and message ID")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApprove(msig, txid, from)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("transfer approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigCancelCmd = &cli.Command{
	Name:      "cancel",
	Usage:     "Cancel a multisig message",
	ArgsUsage: "<multisigAddress messageId> [destination value [methodId methodParams]]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the cancel message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 2 {
			return fmt.Errorf("must pass at least multisig address and message ID")
		}

		if cctx.Args().Len() > 2 && cctx.Args().Len() < 4 {
			return fmt.Errorf("usage: msig cancel <msig addr> <message ID> <desination> <value>")
		}

		if cctx.Args().Len() > 4 && cctx.Args().Len() != 6 {
			return fmt.Errorf("usage: msig cancel <msig addr> <message ID> <desination> <value> [ <method> <params> ]")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		var msgCid cid.Cid
		if cctx.Args().Len() == 2 {
			proto, err := msiger.MsigCancel(msig, txid, from)
			if err != nil {
				return err
			}

			msgCid, err = send(nk, proto)
			if err != nil {
				log.Error(err)
				return err
			}
		} else {
			dest, err := address.NewFromString(cctx.Args().Get(2))
			if err != nil {
				return err
			}

			value, err := types.ParseFIL(cctx.Args().Get(3))
			if err != nil {
				return err
			}

			var method uint64
			var params []byte
			if cctx.Args().Len() == 6 {
				m, err := strconv.ParseUint(cctx.Args().Get(4), 10, 64)
				if err != nil {
					return err
				}
				method = m

				p, err := hex.DecodeString(cctx.Args().Get(5))
				if err != nil {
					return err
				}
				params = p
			}

			proto, err := msiger.MsigCancelTxnHash(msig, txid, dest, types.BigInt(value), from, method, params)
			if err != nil {
				return err
			}

			msgCid, err = send(nk, proto)
			if err != nil {
				log.Error(err)
				return err
			}
		}

		fmt.Println("sent cancel in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigRemoveProposeCmd = &cli.Command{
	Name:      "remove-propose",
	Usage:     "Propose to remove a signer",
	ArgsUsage: "[multisigAddress signer]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "decrease-threshold",
			Usage: "whether the number of required signers should be decreased",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the propose message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("must pass multisig address and signer address")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		addr, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigRemoveSigner(msig, from, addr, cctx.Bool("decrease-threshold"))
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent remove proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigAddProposeCmd = &cli.Command{
	Name:      "add-propose",
	Usage:     "Propose to add a signer",
	ArgsUsage: "[multisigAddress signer]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "increase-threshold",
			Usage: "whether the number of required signers should be increased",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the propose message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("must pass multisig address and signer address")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		addr, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigAddPropose(msig, from, addr, cctx.Bool("increase-threshold"))
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "sent add proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigAddApproveCmd = &cli.Command{
	Name:      "add-approve",
	Usage:     "Approve a message to add a signer",
	ArgsUsage: "[multisigAddress proposerAddress txId newAddress increaseThreshold]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 5 {
			return fmt.Errorf("must pass multisig address, proposer address, transaction id, new signer address, whether to increase threshold")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		prop, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		newAdd, err := address.NewFromString(cctx.Args().Get(3))
		if err != nil {
			return err
		}

		inc, err := strconv.ParseBool(cctx.Args().Get(4))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigAddApprove(msig, from, txid, prop, newAdd, inc)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent add approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigAddCancelCmd = &cli.Command{
	Name:      "add-cancel",
	Usage:     "Cancel a message to add a signer",
	ArgsUsage: "[multisigAddress txId newAddress increaseThreshold]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 4 {
			return fmt.Errorf("must pass multisig address, transaction id, new signer address, whether to increase threshold")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		newAdd, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		inc, err := strconv.ParseBool(cctx.Args().Get(3))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigAddCancel(msig, from, txid, newAdd, inc)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent add cancellation in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigSwapProposeCmd = &cli.Command{
	Name:      "swap-propose",
	Usage:     "Propose to swap signers",
	ArgsUsage: "[multisigAddress oldAddress newAddress]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 3 {
			return fmt.Errorf("must pass multisig address, old signer address, new signer address")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		oldAdd, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		newAdd, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigSwapPropose(msig, from, oldAdd, newAdd)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}
		fmt.Println("sent swap proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigSwapApproveCmd = &cli.Command{
	Name:      "swap-approve",
	Usage:     "Approve a message to swap signers",
	ArgsUsage: "[multisigAddress proposerAddress txId oldAddress newAddress]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 5 {
			return fmt.Errorf("must pass multisig address, proposer address, transaction id, old signer address, new signer address")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		prop, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		oldAdd, err := address.NewFromString(cctx.Args().Get(3))
		if err != nil {
			return err
		}

		newAdd, err := address.NewFromString(cctx.Args().Get(4))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigSwapApprove(msig, from, txid, prop, oldAdd, newAdd)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent swap approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigSwapCancelCmd = &cli.Command{
	Name:      "swap-cancel",
	Usage:     "Cancel a message to swap signers",
	ArgsUsage: "[multisigAddress txId oldAddress newAddress]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 4 {
			return fmt.Errorf("must pass multisig address, transaction id, old signer address, new signer address")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		oldAdd, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		newAdd, err := address.NewFromString(cctx.Args().Get(3))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigSwapCancel(msig, from, txid, oldAdd, newAdd)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent swap cancellation in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigLockProposeCmd = &cli.Command{
	Name:      "lock-propose",
	Usage:     "Propose to lock up some balance",
	ArgsUsage: "[multisigAddress startEpoch unlockDuration amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the propose message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 4 {
			return fmt.Errorf("must pass multisig address, start epoch, unlock duration, and amount")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		start, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		duration, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		amount, err := types.ParseFIL(cctx.Args().Get(3))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		params, actErr := actors.SerializeParams(&msig2.LockBalanceParams{
			StartEpoch:     abi.ChainEpoch(start),
			UnlockDuration: abi.ChainEpoch(duration),
			Amount:         big.Int(amount),
		})

		if actErr != nil {
			return actErr
		}

		proto, err := msiger.MsigPropose(msig, msig, big.Zero(), from, uint64(multisig.Methods.LockBalance), params)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}
		fmt.Println("sent lock proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigLockApproveCmd = &cli.Command{
	Name:      "lock-approve",
	Usage:     "Approve a message to lock up some balance",
	ArgsUsage: "[multisigAddress proposerAddress txId startEpoch unlockDuration amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 6 {
			return fmt.Errorf("must pass multisig address, proposer address, tx id, start epoch, unlock duration, and amount")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		prop, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		start, err := strconv.ParseUint(cctx.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}

		duration, err := strconv.ParseUint(cctx.Args().Get(4), 10, 64)
		if err != nil {
			return err
		}

		amount, err := types.ParseFIL(cctx.Args().Get(5))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		params, actErr := actors.SerializeParams(&msig2.LockBalanceParams{
			StartEpoch:     abi.ChainEpoch(start),
			UnlockDuration: abi.ChainEpoch(duration),
			Amount:         big.Int(amount),
		})

		if actErr != nil {
			return actErr
		}

		proto, err := msiger.MsigApproveTxnHash(msig, txid, prop, msig, big.Zero(), from, uint64(multisig.Methods.LockBalance), params)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent lock approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigLockCancelCmd = &cli.Command{
	Name:      "lock-cancel",
	Usage:     "Cancel a message to lock up some balance",
	ArgsUsage: "[multisigAddress txId startEpoch unlockDuration amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the cancel message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 5 {
			return fmt.Errorf("must pass multisig address, tx id, start epoch, unlock duration, and amount")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		start, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		duration, err := strconv.ParseUint(cctx.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}

		amount, err := types.ParseFIL(cctx.Args().Get(4))
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		params, actErr := actors.SerializeParams(&msig2.LockBalanceParams{
			StartEpoch:     abi.ChainEpoch(start),
			UnlockDuration: abi.ChainEpoch(duration),
			Amount:         big.Int(amount),
		})

		if actErr != nil {
			return actErr
		}

		proto, err := msiger.MsigCancelTxnHash(msig, txid, msig, big.Zero(), from, uint64(multisig.Methods.LockBalance), params)
		if err != nil {
			return err
		}
		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent lock cancellation in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigThresholdProposeCmd = &cli.Command{
	Name:      "threshold-propose",
	Usage:     "Propose setting a different signing threshold on the account",
	ArgsUsage: "<multisigAddress newM>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the proposal from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("must pass multisig address and new threshold value")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		newM, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		var from address.Address
		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		params, actErr := actors.SerializeParams(&msig2.ChangeNumApprovalsThresholdParams{
			NewThreshold: newM,
		})

		if actErr != nil {
			return actErr
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(msig, msig, big.Zero(), from, uint64(multisig.Methods.ChangeNumApprovalsThreshold), params)
		if err != nil {
			return fmt.Errorf("failed to propose change of threshold: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent change threshold proposal in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigThresholdApproveCmd = &cli.Command{
	Name:      "approve-threshold",
	Usage:     "Approve a message to setting a different signing threshold on the account",
	ArgsUsage: "[multisigAddress proposerAddress txId newM]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the approve message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 4 {
			return fmt.Errorf("must pass multisig address, proposer address, transaction id, newM")
		}

		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		prop, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		newM, err := strconv.ParseUint(cctx.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}

		var from address.Address

		f, err := address.NewFromString(cctx.String("from"))
		if err != nil {
			return err
		}
		from = f

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		params, actErr := actors.SerializeParams(&msig2.ChangeNumApprovalsThresholdParams{
			NewThreshold: newM,
		})

		if actErr != nil {
			return actErr
		}

		proto, err := msiger.MsigApproveTxnHash(msig, txid, prop, msig, big.Zero(), from, uint64(multisig.Methods.ChangeNumApprovalsThreshold), params)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent change threshold approval in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigWithdrawBalanceProposeCmd = &cli.Command{
	Name:  "withdraw-propose",
	Usage: "Propose to withdraw FIL from the miner",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "specify address to send message from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multisig",
			Usage:    "specify multisig that will receive the message",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "specify miner being acted upon",
			Required: true,
		},
	},
	ArgsUsage: "[amount]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass amount to withdraw")
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		val, err := types.ParseFIL(cctx.Args().First())
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&miner5.WithdrawBalanceParams{
			AmountRequested: abi.TokenAmount(val),
		})
		if err != nil {
			return err
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.WithdrawBalance), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "withdraw propose message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigWithdrawBalanceApproveCmd = &cli.Command{
	Name:  "withdraw-approve",
	Usage: "Approve to withdraw FIL from the miner",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "specify address to send message from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multisig",
			Usage:    "specify multisig that will receive the message",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "specify miner being acted upon",
			Required: true,
		},
	},
	ArgsUsage: "[amount txnId proposer]",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 3 {
			return fmt.Errorf("must pass amount, txn Id, and proposer address")
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		val, err := types.ParseFIL(cctx.Args().First())
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&miner5.WithdrawBalanceParams{
			AmountRequested: abi.TokenAmount(val),
		})
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		proposer, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApproveTxnHash(multisigAddr, txid, proposer, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.WithdrawBalance), sp)
		if err != nil {
			return xerrors.Errorf("approving message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "withdraw approve message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigChangeOwnerProposeCmd = &cli.Command{
	Name:  "change-owner-propose",
	Usage: "Propose an owner address change",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "specify address to send message from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multisig",
			Usage:    "specify multisig that will receive the message",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "specify miner being acted upon",
			Required: true,
		},
	},
	ArgsUsage: "[newOwner]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass new owner address")
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		conf := config.Conf()
		newAddrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
		if err != nil {
			return err
		}
		newAddr, err := address.NewFromString(newAddrStr)
		if err != nil {
			return err
		}

		owner, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		if owner == newAddrStr {
			return fmt.Errorf("owner address already set to %s", na)
		}

		sp, err := actors.SerializeParams(&newAddr)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeOwnerAddress), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change owner propose message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigChangeOwnerApproveCmd = &cli.Command{
	Name:  "change-owner-approve",
	Usage: "Approve an owner address change",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "specify address to send message from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multisig",
			Usage:    "specify multisig that will receive the message",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "specify miner being acted upon",
			Required: true,
		},
	},
	ArgsUsage: "[newOwner txnId proposer]",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 3 {
			return fmt.Errorf("must pass new owner address, txn Id, and proposer address")
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		conf := config.Conf()
		newAddrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
		if err != nil {
			return err
		}
		newAddr, err := address.NewFromString(newAddrStr)
		if err != nil {
			return err
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		proposer, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		owner, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		if owner == newAddrStr {
			return fmt.Errorf("owner address already set to %s", na)
		}

		sp, err := actors.SerializeParams(&newAddr)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApproveTxnHash(multisigAddr, txid, proposer, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeOwnerAddress), sp)
		if err != nil {
			return xerrors.Errorf("approving message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change owner approve message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

func getInputs(cctx *cli.Context) (address.Address, address.Address, address.Address, error) {
	multisigAddr, err := address.NewFromString(cctx.String("multisig"))
	if err != nil {
		return address.Undef, address.Undef, address.Undef, err
	}

	sender, err := address.NewFromString(cctx.String("from"))
	if err != nil {
		return address.Undef, address.Undef, address.Undef, err
	}

	minerAddr, err := address.NewFromString(cctx.String("miner"))
	if err != nil {
		return address.Undef, address.Undef, address.Undef, err
	}

	return multisigAddr, sender, minerAddr, nil
}

func waitProposalMsg(msgCidStr string) error {
	fmt.Println("message waiting for confirmation...")
	conf := config.Conf()

	var wait *client.MsgLookup
	var err error
	var i int = 0
	for {
		if i > 60 {
			return xerrors.New("wait timeout")
		}

		time.Sleep(30 * time.Second)
		wait, err = client.LotusStateSearchMsg(conf.Chain.RpcAddr, conf.Chain.Token, msgCidStr)
		if err != nil {
			log.Error(err)
			return err
		}
		if wait == nil {
			i++
			continue
		}

		break
	}

	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("propose returned exit %d", wait.Receipt.ExitCode)
	}

	var ret multisig.ProposeReturn
	err = ret.UnmarshalCBOR(bytes.NewReader(wait.Receipt.Return))
	if err != nil {
		return xerrors.Errorf("decoding proposal return: %w", err)
	}

	fmt.Printf("TxnID: %d ", ret.TxnID)
	return nil
}

func waitMsg(msgCidStr string) error {
	fmt.Println("message waiting for confirmation...")
	conf := config.Conf()
	var wait *client.MsgLookup
	var err error
	var i int = 0
	for {
		if i > 60 {
			return xerrors.New("wait timeout")
		}

		time.Sleep(30 * time.Second)
		wait, err = client.LotusStateSearchMsg(conf.Chain.RpcAddr, conf.Chain.Token, msgCidStr)
		if err != nil {
			log.Error(err)
			return err
		}
		if wait == nil {
			i++
			continue
		}

		break
	}

	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("msg returned exit %d", wait.Receipt.ExitCode)
	}

	fmt.Println("message confirm!!!")

	return nil
}

// ------------------------------------

type msig struct{}

func NewMsiger() *msig {
	return &msig{}
}

func (m *msig) messageBuilder(from address.Address) (multisig.MessageBuilder, error) {
	av, err := actors.VersionForNetwork(network.Version16)
	if err != nil {
		return nil, err
	}

	return multisig.Message(av, from), nil
}

func (m *msig) MsigCreate(req uint64, addrs []address.Address, duration abi.ChainEpoch, val types.BigInt, src address.Address, gp types.BigInt) (*types.Message, error) {
	mb, err := m.messageBuilder(src)
	if err != nil {
		return nil, err
	}

	msg, err := mb.Create(addrs, req, 0, duration, val)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (m *msig) MsigPropose(msig address.Address, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (*types.Message, error) {

	mb, err := m.messageBuilder(src)
	if err != nil {
		return nil, err
	}

	msg, err := mb.Propose(msig, to, amt, abi.MethodNum(method), params)
	if err != nil {
		return nil, xerrors.Errorf("failed to create proposal: %w", err)
	}

	return msg, nil
}

func (m *msig) MsigAddPropose(msig address.Address, src address.Address, newAdd address.Address, inc bool) (*types.Message, error) {
	enc, actErr := serializeAddParams(newAdd, inc)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigPropose(msig, msig, big.Zero(), src, uint64(multisig.Methods.AddSigner), enc)
}

func (m *msig) MsigAddApprove(msig address.Address, src address.Address, txID uint64, proposer address.Address, newAdd address.Address, inc bool) (*types.Message, error) {
	enc, actErr := serializeAddParams(newAdd, inc)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigApproveTxnHash(msig, txID, proposer, msig, big.Zero(), src, uint64(multisig.Methods.AddSigner), enc)
}

func (m *msig) MsigAddCancel(msig address.Address, src address.Address, txID uint64, newAdd address.Address, inc bool) (*types.Message, error) {
	enc, actErr := serializeAddParams(newAdd, inc)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigCancelTxnHash(msig, txID, msig, big.Zero(), src, uint64(multisig.Methods.AddSigner), enc)
}

func (m *msig) MsigSwapPropose(msig address.Address, src address.Address, oldAdd address.Address, newAdd address.Address) (*types.Message, error) {
	enc, actErr := serializeSwapParams(oldAdd, newAdd)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigPropose(msig, msig, big.Zero(), src, uint64(multisig.Methods.SwapSigner), enc)
}

func (m *msig) MsigSwapApprove(msig address.Address, src address.Address, txID uint64, proposer address.Address, oldAdd address.Address, newAdd address.Address) (*types.Message, error) {
	enc, actErr := serializeSwapParams(oldAdd, newAdd)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigApproveTxnHash(msig, txID, proposer, msig, big.Zero(), src, uint64(multisig.Methods.SwapSigner), enc)
}

func (m *msig) MsigSwapCancel(msig address.Address, src address.Address, txID uint64, oldAdd address.Address, newAdd address.Address) (*types.Message, error) {
	enc, actErr := serializeSwapParams(oldAdd, newAdd)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigCancelTxnHash(msig, txID, msig, big.Zero(), src, uint64(multisig.Methods.SwapSigner), enc)
}

func (m *msig) MsigApprove(msig address.Address, txID uint64, src address.Address) (*types.Message, error) {
	return m.MsigApproveOrCancelSimple(api.MsigApprove, msig, txID, src)
}

func (m *msig) MsigApproveTxnHash(msig address.Address, txID uint64, proposer address.Address, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (*types.Message, error) {
	return m.MsigApproveOrCancelTxnHash(api.MsigApprove, msig, txID, proposer, to, amt, src, method, params)
}

func (m *msig) MsigCancel(msig address.Address, txID uint64, src address.Address) (*types.Message, error) {
	return m.MsigApproveOrCancelSimple(api.MsigCancel, msig, txID, src)
}

func (m *msig) MsigCancelTxnHash(msig address.Address, txID uint64, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (*types.Message, error) {
	return m.MsigApproveOrCancelTxnHash(api.MsigCancel, msig, txID, src, to, amt, src, method, params)
}

func (m *msig) MsigRemoveSigner(msig address.Address, proposer address.Address, toRemove address.Address, decrease bool) (*types.Message, error) {
	enc, actErr := serializeRemoveParams(toRemove, decrease)
	if actErr != nil {
		return nil, actErr
	}

	return m.MsigPropose(msig, msig, types.NewInt(0), proposer, uint64(multisig.Methods.RemoveSigner), enc)
}

func (m *msig) MsigApproveOrCancelSimple(operation api.MsigProposeResponse, msig address.Address, txID uint64, src address.Address) (*types.Message, error) {
	if msig == address.Undef {
		return nil, xerrors.Errorf("must provide multisig address")
	}

	if src == address.Undef {
		return nil, xerrors.Errorf("must provide source address")
	}

	mb, err := m.messageBuilder(src)
	if err != nil {
		return nil, err
	}

	var msg *types.Message
	switch operation {
	case api.MsigApprove:
		msg, err = mb.Approve(msig, txID, nil)
	case api.MsigCancel:
		msg, err = mb.Cancel(msig, txID, nil)
	default:
		return nil, xerrors.Errorf("Invalid operation for msigApproveOrCancel")
	}
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (m *msig) MsigApproveOrCancelTxnHash(operation api.MsigProposeResponse, msig address.Address, txID uint64, proposer address.Address, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (*types.Message, error) {
	if msig == address.Undef {
		return nil, xerrors.Errorf("must provide multisig address")
	}

	if src == address.Undef {
		return nil, xerrors.Errorf("must provide source address")
	}

	if proposer.Protocol() != address.ID {
		proposerID, err := client.LotusStateLookupID(config.Conf().Chain.RpcAddr, config.Conf().Chain.Token, proposer.String())
		if err != nil {
			return nil, err
		}
		proposer, err = address.NewFromString(proposerID)
		if err != nil {
			return nil, err
		}
	}

	p := multisig.ProposalHashData{
		Requester: proposer,
		To:        to,
		Value:     amt,
		Method:    abi.MethodNum(method),
		Params:    params,
	}

	mb, err := m.messageBuilder(src)
	if err != nil {
		return nil, err
	}

	var msg *types.Message
	switch operation {
	case api.MsigApprove:
		msg, err = mb.Approve(msig, txID, &p)
	case api.MsigCancel:
		msg, err = mb.Cancel(msig, txID, &p)
	default:
		return nil, xerrors.Errorf("Invalid operation for msigApproveOrCancel")
	}
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func serializeAddParams(new address.Address, inc bool) ([]byte, error) {
	enc, actErr := actors.SerializeParams(&msig2.AddSignerParams{
		Signer:   new,
		Increase: inc,
	})
	if actErr != nil {
		return nil, actErr
	}

	return enc, nil
}

func serializeSwapParams(old address.Address, new address.Address) ([]byte, error) {
	enc, actErr := actors.SerializeParams(&msig2.SwapSignerParams{
		From: old,
		To:   new,
	})
	if actErr != nil {
		return nil, actErr
	}

	return enc, nil
}

func serializeRemoveParams(rem address.Address, dec bool) ([]byte, error) {
	enc, actErr := actors.SerializeParams(&msig2.RemoveSignerParams{
		Signer:   rem,
		Decrease: dec,
	})
	if actErr != nil {
		return nil, actErr
	}

	return enc, nil
}
