package wallet

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	bt2 "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/types"
	init2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	msig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

const emptyWorker = "<empty>"

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
		&cli.StringFlag{
			Name:  "conf-path",
			Usage: "config.yaml path",
			Value: "",
		},
	},
	Subcommands: []*cli.Command{
		msigCreateCmd,
		msigInspectCmd,
		msigProposeCmd,
		msigRemoveProposeCmd,
		msigApproveCmd,
		msigCancelCmd,
		msigTransferProposeCmd,
		msigTransferApproveCmd,
		msigTransferCancelCmd,
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
		msigChangeWorkerProposeCmd,
		msigChangeWorkerApproveCmd,
		msigConfirmChangeWorkerProposeCmd,
		msigConfirmChangeWorkerApproveCmd,
		msigSetControlProposeCmd,
		msigSetControlApproveCmd,
		msigProposeChangeBeneficiary,
		msigConfirmChangeBeneficiary,
	},
	Before: func(c *cli.Context) error {
		config.InitConfig(c.String("conf-path"))
		return nil
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

var msigInspectCmd = &cli.Command{
	Name:      "inspect",
	Usage:     "Inspect a multisig wallet",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "vesting",
			Usage: "Include vesting details",
		},
		&cli.BoolFlag{
			Name:  "decode-params",
			Usage: "Decode parameters of transaction proposals",
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must specify address of multisig to inspect")
		}

		conf := config.Conf()
		api, closer, err := client.NewLotusAPI(conf.Chain.RpcAddr, conf.Chain.Token)
		if err != nil {
			return err
		}
		defer closer()
		ctx := context.Background()

		store := adt.WrapStore(ctx, cbor.NewCborStore(blockstore.NewAPIBlockstore(api)))

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		head, err := api.ChainHead(ctx)
		if err != nil {
			return err
		}

		act, err := api.StateGetActor(ctx, maddr, head.Key())
		if err != nil {
			return err
		}

		ownId, err := api.StateLookupID(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		mstate, err := multisig.Load(store, act)
		if err != nil {
			return err
		}
		locked, err := mstate.LockedBalance(head.Height())
		if err != nil {
			return err
		}

		fmt.Fprintf(cctx.App.Writer, "Balance: %s\n", types.FIL(act.Balance))
		fmt.Fprintf(cctx.App.Writer, "Spendable: %s\n", types.FIL(types.BigSub(act.Balance, locked)))

		if cctx.Bool("vesting") {
			ib, err := mstate.InitialBalance()
			if err != nil {
				return err
			}
			fmt.Fprintf(cctx.App.Writer, "InitialBalance: %s\n", types.FIL(ib))
			se, err := mstate.StartEpoch()
			if err != nil {
				return err
			}
			fmt.Fprintf(cctx.App.Writer, "StartEpoch: %d\n", se)
			ud, err := mstate.UnlockDuration()
			if err != nil {
				return err
			}
			fmt.Fprintf(cctx.App.Writer, "UnlockDuration: %d\n", ud)
		}

		signers, err := mstate.Signers()
		if err != nil {
			return err
		}
		threshold, err := mstate.Threshold()
		if err != nil {
			return err
		}
		fmt.Fprintf(cctx.App.Writer, "Threshold: %d / %d\n", threshold, len(signers))
		fmt.Fprintln(cctx.App.Writer, "Signers:")

		signerTable := tabwriter.NewWriter(cctx.App.Writer, 8, 4, 2, ' ', 0)
		fmt.Fprintf(signerTable, "ID\tAddress\n")
		for _, s := range signers {
			signerActor, err := api.StateAccountKey(ctx, s, types.EmptyTSK)
			if err != nil {
				fmt.Fprintf(signerTable, "%s\t%s\n", s, "N/A")
			} else {
				fmt.Fprintf(signerTable, "%s\t%s\n", s, signerActor)
			}
		}
		if err := signerTable.Flush(); err != nil {
			return xerrors.Errorf("flushing output: %+v", err)
		}

		pending := make(map[int64]multisig.Transaction)
		if err := mstate.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
			pending[id] = txn
			return nil
		}); err != nil {
			return xerrors.Errorf("reading pending transactions: %w", err)
		}

		decParams := cctx.Bool("decode-params")
		fmt.Fprintln(cctx.App.Writer, "Transactions: ", len(pending))
		if len(pending) > 0 {
			var txids []int64
			for txid := range pending {
				txids = append(txids, txid)
			}
			sort.Slice(txids, func(i, j int) bool {
				return txids[i] < txids[j]
			})

			w := tabwriter.NewWriter(cctx.App.Writer, 8, 4, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tState\tApprovals\tTo\tValue\tMethod\tParams\n")
			for _, txid := range txids {
				tx := pending[txid]
				target := tx.To.String()
				if tx.To == ownId {
					target += " (self)"
				}
				targAct, err := api.StateGetActor(ctx, tx.To, types.EmptyTSK)
				paramStr := fmt.Sprintf("%x", tx.Params)

				if err != nil {
					if tx.Method == 0 {
						fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s(%d)\t%s\n", txid, "pending", len(tx.Approved), target, types.FIL(tx.Value), "Send", tx.Method, paramStr)
					} else {
						fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s(%d)\t%s\n", txid, "pending", len(tx.Approved), target, types.FIL(tx.Value), "new account, unknown method", tx.Method, paramStr)
					}
				} else {
					// todo 反解params
					method := consensus.NewActorRegistry().Methods[targAct.Code][tx.Method] // TODO: use remote map

					if decParams && tx.Method != 0 {
						ptyp := reflect.New(method.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
						if err := ptyp.UnmarshalCBOR(bytes.NewReader(tx.Params)); err != nil {
							return xerrors.Errorf("failed to decode parameters of transaction %d: %w", txid, err)
						}

						b, err := json.Marshal(ptyp)
						if err != nil {
							return xerrors.Errorf("could not json marshal parameter type: %w", err)
						}

						paramStr = string(b)
					}

					fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s(%d)\t%s\n", txid, "pending", len(tx.Approved), target, types.FIL(tx.Value), method.Name, tx.Method, paramStr)
				}
			}
			if err := w.Flush(); err != nil {
				return xerrors.Errorf("flushing output: %+v", err)
			}

		}

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
	ArgsUsage: "<multisigAddress txId> [proposerAddress destination value [methodId methodParams]]",
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
	ArgsUsage: "<multisigAddress txId>",
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

var msigTransferCancelCmd = &cli.Command{
	Name:      "transfer-cancel",
	Usage:     "Cancel transfer multisig message",
	ArgsUsage: "<multisigAddress txId> [destination value [methodId methodParams]]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "account to send the cancel message from",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("must have multisig address and txId")
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

		proto, err := msiger.MsigCancel(msig, txid, from)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("sent transfer cancel in message: ", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigCancelCmd = &cli.Command{
	Name:      "cancel",
	Usage:     "Cancel a multisig message",
	ArgsUsage: "<multisigAddress txId> [destination value [methodId methodParams]]",
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

		owner, _, _, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
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

		owner, _, _, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
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

var msigChangeWorkerProposeCmd = &cli.Command{
	Name:  "change-worker-propose",
	Usage: "Propose an worker address change",
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
	ArgsUsage: "[newWorker]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass new worker address")
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
		_, workerStr, newWorkerStr, workerChangeEpoch, controlAddressesStr, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		newAddrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
		if err != nil {
			return err
		}

		newAddr, err := address.NewFromString(newAddrStr)
		if err != nil {
			return err
		}

		if newWorkerStr == emptyWorker {
			if workerStr == newAddrStr {
				return fmt.Errorf("worker address already set to %s", na)
			}
		} else {
			if newWorkerStr == newAddrStr {
				fmt.Fprintf(cctx.App.Writer, "Worker key change to %s successfully proposed.\n", na)
				fmt.Fprintf(cctx.App.Writer, "Call 'confirm-change-worker' at or after height %f to complete.\n", workerChangeEpoch)
				return fmt.Errorf("change to worker address %s already pending", na)
			}
		}

		var controlAddresses []address.Address
		for _, addrStr := range controlAddressesStr {
			addr, err := address.NewFromString(addrStr.(string))
			if err != nil {
				return err
			}
			controlAddresses = append(controlAddresses, addr)
		}

		cwp := &miner5.ChangeWorkerAddressParams{
			NewWorker:       newAddr,
			NewControlAddrs: controlAddresses,
		}

		fmt.Fprintf(cctx.App.Writer, "newAddr: %s\n", newAddr)
		fmt.Fprintf(cctx.App.Writer, "NewControlAddrs: %s\n", controlAddresses)

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeWorkerAddress), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change worker propose message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigChangeWorkerApproveCmd = &cli.Command{
	Name:  "change-worker-approve",
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
	ArgsUsage: "[newWorker txnId proposer]",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 3 {
			return fmt.Errorf("must have newWorker, txn Id, and proposer address")
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
		newWorker, err := address.NewFromString(newAddrStr)
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

		_, workerStr, newWorkerStr, _, controlAddressesStr, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())

		if newWorkerStr == emptyWorker {
			if workerStr == newAddrStr {
				return fmt.Errorf("worker address already set to %s", na)
			}
		} else {
			if newWorkerStr == newAddrStr {
				fmt.Fprintf(cctx.App.Writer, "Worker key change to %s successfully proposed.\n", na)
				return fmt.Errorf("change to worker address %s already pending", na)
			}
		}

		var controlAddresses []address.Address
		for _, addrStr := range controlAddressesStr {
			addr, err := address.NewFromString(addrStr.(string))
			if err != nil {
				return err
			}
			controlAddresses = append(controlAddresses, addr)
		}

		cwp := &miner5.ChangeWorkerAddressParams{
			NewWorker:       newWorker,
			NewControlAddrs: controlAddresses,
		}

		fmt.Fprintf(cctx.App.Writer, "newAddr: %s\n", newWorker)
		fmt.Fprintf(cctx.App.Writer, "NewControlAddrs: %s\n", controlAddresses)

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApproveTxnHash(multisigAddr, txid, proposer, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeWorkerAddress), sp)
		if err != nil {
			return xerrors.Errorf("approving message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change worker approve message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigConfirmChangeWorkerProposeCmd = &cli.Command{
	Name:      "confirm-change-worker-propose",
	Usage:     "Confirm an worker address change",
	ArgsUsage: "[newWorker]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass new worker address")
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
		_, _, newWorkerStr, workerChangeEpoch, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		newAddrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
		if err != nil {
			return err
		}

		newAddr, err := address.NewFromString(newAddrStr)
		if err != nil {
			return err
		}

		if newWorkerStr == emptyWorker {
			return xerrors.Errorf("no worker key change proposed")
		} else if newWorkerStr != newAddrStr {
			return xerrors.Errorf("worker key %s does not match current worker key proposal %s", newAddr, newWorkerStr)
		}

		height, err := client.LotusChainHead(conf.Chain.RpcAddr, conf.Chain.Token)
		if err != nil {
			return xerrors.Errorf("failed to get the chain head: %w", err)
		} else if height < workerChangeEpoch {
			return xerrors.Errorf("worker key change cannot be confirmed until %d, current height is %d", workerChangeEpoch, height)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ConfirmChangeWorkerAddress), nil)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "confirm worker propose message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigConfirmChangeWorkerApproveCmd = &cli.Command{
	Name:  "confirm-change-worker-approve",
	Usage: "Confirm an worker address change",
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
	ArgsUsage: "[newWorker txnId proposer]",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 3 {
			return fmt.Errorf("must have newWorker, txn Id, and proposer address")
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

		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		proposer, err := address.NewFromString(cctx.Args().Get(2))
		if err != nil {
			return err
		}

		_, _, newWorkerStr, workerChangeEpoch, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())

		if newWorkerStr == emptyWorker {
			return xerrors.Errorf("no worker key change proposed")
		} else if newWorkerStr != newAddrStr {
			return xerrors.Errorf("worker key %s does not match current worker key proposal %s", newAddrStr, newWorkerStr)
		}

		height, err := client.LotusChainHead(conf.Chain.RpcAddr, conf.Chain.Token)
		if err != nil {
			return xerrors.Errorf("failed to get the chain head: %w", err)
		} else if height < workerChangeEpoch {
			return xerrors.Errorf("worker key change cannot be confirmed until %d, current height is %d", workerChangeEpoch, height)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApproveTxnHash(multisigAddr, txid, proposer, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ConfirmChangeWorkerAddress), nil)
		if err != nil {
			return xerrors.Errorf("approving message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change worker approve message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigSetControlProposeCmd = &cli.Command{
	Name:  "set-control-propose",
	Usage: "set control address(-es) propose",
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
	ArgsUsage: "[...address]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass new owner address")
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		conf := config.Conf()
		_, workerStr, _, _, controlAddressesStr, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		worker, err := address.NewFromString(workerStr)
		if err != nil {
			return err
		}

		del := map[address.Address]struct{}{}
		existing := map[address.Address]struct{}{}
		for _, controlAddress := range controlAddressesStr {
			kaStr, err := client.LotusStateAccountKey(conf.Chain.RpcAddr, conf.Chain.Token, controlAddress.(string))
			if err != nil {
				return err
			}

			ka, err := address.NewFromString(kaStr)
			if err != nil {
				return err
			}

			del[ka] = struct{}{}
			existing[ka] = struct{}{}
		}

		var toSet []address.Address

		for i, as := range cctx.Args().Slice() {
			na, err := address.NewFromString(as)
			if err != nil {
				return xerrors.Errorf("parsing address %d: %w", i, err)
			}

			addrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
			if err != nil {
				return xerrors.Errorf("looking up %s: %w", addrStr, err)
			}

			ka, err := address.NewFromString(addrStr)
			if err != nil {
				return xerrors.Errorf("parsing address %d: %w", i, err)
			}

			delete(del, ka)
			toSet = append(toSet, ka)
		}

		for a := range del {
			fmt.Println("Remove", a)
		}
		for _, a := range toSet {
			if _, exists := existing[a]; !exists {
				fmt.Println("Add", a)
			}
		}

		cwp := &miner5.ChangeWorkerAddressParams{
			NewWorker:       worker,
			NewControlAddrs: toSet,
		}

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeWorkerAddress), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "change control address propose message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigSetControlApproveCmd = &cli.Command{
	Name:  "set-control-approve",
	Usage: "set control address(-es) approve",
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
	ArgsUsage: "[txnId proposer ...address]",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() == 0 {
			return fmt.Errorf("must have txn Id, and proposer address and ...address")
		}

		txid, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}

		proposer, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		conf := config.Conf()
		_, workerStr, _, _, controlAddressesStr, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, minerAddr.String())
		if err != nil {
			return err
		}

		worker, err := address.NewFromString(workerStr)
		if err != nil {
			return err
		}

		del := map[address.Address]struct{}{}
		existing := map[address.Address]struct{}{}
		for _, controlAddress := range controlAddressesStr {
			kaStr, err := client.LotusStateAccountKey(conf.Chain.RpcAddr, conf.Chain.Token, controlAddress.(string))
			if err != nil {
				return err
			}

			ka, err := address.NewFromString(kaStr)
			if err != nil {
				return err
			}

			del[ka] = struct{}{}
			existing[ka] = struct{}{}
		}

		var toSet []address.Address

		for i, as := range cctx.Args().Slice() {
			if i == 0 || i == 1 {
				continue
			}

			na, err := address.NewFromString(as)
			if err != nil {
				return xerrors.Errorf("parsing address %d: %w", i, err)
			}

			addrStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
			if err != nil {
				return xerrors.Errorf("looking up %s: %w", addrStr, err)
			}

			ka, err := address.NewFromString(addrStr)
			if err != nil {
				return xerrors.Errorf("parsing address %d: %w", i, err)
			}

			delete(del, ka)
			toSet = append(toSet, ka)
		}

		for a := range del {
			fmt.Println("Remove", a)
		}
		for _, a := range toSet {
			if _, exists := existing[a]; !exists {
				fmt.Println("Add", a)
			}
		}

		cwp := &miner5.ChangeWorkerAddressParams{
			NewWorker:       worker,
			NewControlAddrs: toSet,
		}

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msiger := NewMsiger()

		proto, err := msiger.MsigApproveTxnHash(multisigAddr, txid, proposer, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeWorkerAddress), sp)
		if err != nil {
			return xerrors.Errorf("approving message: %w", err)
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Fprintln(cctx.App.Writer, "set control address approve message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitMsg(msgCid.String())
	},
}

var msigProposeChangeBeneficiary = &cli.Command{
	Name:      "propose-change-beneficiary",
	Usage:     "Propose a beneficiary address change",
	ArgsUsage: "[beneficiaryAddress quota expiration]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "overwrite-pending-change",
			Usage: "Overwrite the current beneficiary change proposal",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 3 {
			return fmt.Errorf("must be set: beneficiaryAddress quota expiration")
		}

		conf := config.Conf()
		api, closer, err := client.NewLotusAPI(conf.Chain.RpcAddr, conf.Chain.Token)
		if err != nil {
			return err
		}
		defer closer()
		ctx := context.Background()

		na, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return xerrors.Errorf("parsing beneficiary address: %w", err)
		}

		newAddr, err := api.StateLookupID(ctx, na, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("looking up new beneficiary address: %w", err)
		}

		quota, err := types.ParseFIL(cctx.Args().Get(1))
		if err != nil {
			return xerrors.Errorf("parsing quota: %w", err)
		}

		expiration, err := types.BigFromString(cctx.Args().Get(2))
		if err != nil {
			return xerrors.Errorf("parsing expiration: %w", err)
		}

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		mi, err := api.StateMinerInfo(ctx, minerAddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if mi.PendingBeneficiaryTerm != nil {
			fmt.Println("WARNING: replacing Pending Beneficiary Term of:")
			fmt.Println("Beneficiary: ", mi.PendingBeneficiaryTerm.NewBeneficiary)
			fmt.Println("Quota:", mi.PendingBeneficiaryTerm.NewQuota)
			fmt.Println("Expiration Epoch:", mi.PendingBeneficiaryTerm.NewExpiration)

			if !cctx.Bool("overwrite-pending-change") {
				return fmt.Errorf("must pass --overwrite-pending-change to replace current pending beneficiary change. Please review CAREFULLY")
			}
		}

		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action. Review what you're about to approve CAREFULLY please")
			return nil
		}

		params := &miner.ChangeBeneficiaryParams{
			NewBeneficiary: newAddr,
			NewQuota:       abi.TokenAmount(quota),
			NewExpiration:  abi.ChainEpoch(expiration.Int64()),
		}

		sp, err := actors.SerializeParams(params)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		msiger := NewMsiger()
		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeBeneficiary), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("Propose Message CID: ", msgCid)

		fmt.Fprintln(cctx.App.Writer, "propose beneficiary change message CID:", msgCid)
		fmt.Println(fmt.Sprintf("%s%s", config.Conf().Chain.Explorer, msgCid.String()))

		return waitProposalMsg(msgCid.String())
	},
}

var msigConfirmChangeBeneficiary = &cli.Command{
	Name:      "confirm-change-beneficiary",
	Usage:     "Confirm a beneficiary address change",
	ArgsUsage: "[minerAddress]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return fmt.Errorf("must be set: minerAddress")
		}

		conf := config.Conf()
		api, closer, err := client.NewLotusAPI(conf.Chain.RpcAddr, conf.Chain.Token)
		if err != nil {
			return err
		}
		defer closer()
		ctx := context.Background()

		multisigAddr, sender, minerAddr, err := getInputs(cctx)
		if err != nil {
			return err
		}

		mi, err := api.StateMinerInfo(ctx, minerAddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if mi.PendingBeneficiaryTerm == nil {
			return fmt.Errorf("no pending beneficiary term found for miner %s", minerAddr)
		}

		fmt.Println("Confirming Pending Beneficiary Term of:")
		fmt.Println("Beneficiary: ", mi.PendingBeneficiaryTerm.NewBeneficiary)
		fmt.Println("Quota:", mi.PendingBeneficiaryTerm.NewQuota)
		fmt.Println("Expiration Epoch:", mi.PendingBeneficiaryTerm.NewExpiration)

		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action. Review what you're about to approve CAREFULLY please")
			return nil
		}

		params := &miner.ChangeBeneficiaryParams{
			NewBeneficiary: mi.PendingBeneficiaryTerm.NewBeneficiary,
			NewQuota:       mi.PendingBeneficiaryTerm.NewQuota,
			NewExpiration:  mi.PendingBeneficiaryTerm.NewExpiration,
		}

		sp, err := actors.SerializeParams(params)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		msiger := NewMsiger()
		proto, err := msiger.MsigPropose(multisigAddr, minerAddr, big.Zero(), sender, uint64(builtin.MethodsMiner.ChangeBeneficiary), sp)
		if err != nil {
			return xerrors.Errorf("proposing message: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, proto)
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Println("Confirm Message CID:", msgCid)

		err = waitMsg(msgCid.String())
		if err != nil {
			log.Error(err)
		}

		updatedMinerInfo, err := api.StateMinerInfo(ctx, minerAddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if updatedMinerInfo.PendingBeneficiaryTerm == nil && updatedMinerInfo.Beneficiary == mi.PendingBeneficiaryTerm.NewBeneficiary {
			fmt.Println("Beneficiary address successfully changed")
		} else {
			fmt.Println("Beneficiary address change awaiting additional confirmations")
		}

		return nil
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

	fmt.Printf("txId: %d ", ret.TxnID)
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

	fmt.Println("message confirm!")

	return nil
}

// ------------------------------------

type msig struct{}

func NewMsiger() *msig {
	return &msig{}
}

func (m *msig) messageBuilder(from address.Address) (multisig.MessageBuilder, error) {
	av, err := actorstypes.VersionForNetwork(network.Version22)
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
