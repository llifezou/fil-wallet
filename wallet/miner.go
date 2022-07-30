package wallet

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/tablewriter"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	"github.com/llifezou/fil-wallet/client"
	"github.com/llifezou/fil-wallet/config"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
)

var minerCmd = &cli.Command{
	Name:  "miner",
	Usage: "manipulate the miner actor",
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
		actorWithdrawCmd,
		actorSetOwnerCmd,
		actorControl,
		actorProposeChangeWorker,
		actorConfirmChangeWorker,
	},
}

var actorWithdrawCmd = &cli.Command{
	Name:      "withdraw",
	Usage:     "withdraw available balance",
	ArgsUsage: "[amount (FIL)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		act := cctx.String("actor")
		maddr, err := address.NewFromString(act)
		if err != nil {
			return fmt.Errorf("parsing address %s: %w", act, err)
		}

		conf := config.Conf()
		ownerStr, _, _, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		owner, err := address.NewFromString(ownerStr)
		if err != nil {
			return err
		}

		availableBalance, err := client.LotusStateMinerAvailableBalance(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		available, err := big.FromString(availableBalance)
		if err != nil {
			return err
		}

		amount := available
		if cctx.Args().Present() {
			f, err := types.ParseFIL(cctx.Args().First())
			if err != nil {
				return xerrors.Errorf("parsing 'amount' argument: %w", err)
			}

			amount = abi.TokenAmount(f)

			if amount.GreaterThan(available) {
				return xerrors.Errorf("can't withdraw more funds than available; requested: %s; available: %s", types.FIL(amount), types.FIL(available))
			}
		}

		params, err := actors.SerializeParams(&miner5.WithdrawBalanceParams{
			AmountRequested: amount, // Default to attempting to withdraw all the extra funds in the miner actor
		})
		if err != nil {
			return err
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, &types.Message{
			To:     maddr,
			From:   owner,
			Value:  types.NewInt(0),
			Method: builtin.MethodsMiner.WithdrawBalance,
			Params: params,
		})
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Printf("Requested rewards withdrawal in message %s\n", msgCid.String())

		return waitMsg(msgCid.String())
	},
}

var actorSetOwnerCmd = &cli.Command{
	Name:      "set-owner",
	Usage:     "Set owner address (this command should be invoked twice, first with the old owner as the senderAddress, and then with the new owner)",
	ArgsUsage: "[newOwnerAddress senderAddress]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		if cctx.NArg() != 2 {
			return fmt.Errorf("must pass new owner address and sender address")
		}

		act := cctx.String("actor")
		maddr, err := address.NewFromString(act)
		if err != nil {
			return fmt.Errorf("parsing address %s: %w", act, err)
		}

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		conf := config.Conf()
		newAddrIdStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, na.String())
		if err != nil {
			return err
		}

		newAddrId, err := address.NewFromString(newAddrIdStr)
		if err != nil {
			return err
		}

		fa, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		fromAddrIdStr, err := client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, fa.String())
		if err != nil {
			return err
		}

		fromAddrId, err := address.NewFromString(fromAddrIdStr)
		if err != nil {
			return err
		}

		ownerStr, _, _, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		if fromAddrIdStr != ownerStr && fromAddrId != newAddrId {
			return xerrors.New("from address must either be the old owner or the new owner")
		}

		sp, err := actors.SerializeParams(&newAddrId)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, &types.Message{
			From:   fromAddrId,
			To:     maddr,
			Method: builtin.MethodsMiner.ChangeOwnerAddress,
			Value:  big.Zero(),
			Params: sp,
		})
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Printf("Message CID: %s\n", msgCid.String())

		return waitMsg(msgCid.String())
	},
}

var actorControl = &cli.Command{
	Name:  "control",
	Usage: "Manage control addresses",
	Subcommands: []*cli.Command{
		actorControlList,
		actorControlSet,
	},
}

var actorControlList = &cli.Command{
	Name:  "list",
	Usage: "Get currently set control addresses",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
		&cli.BoolFlag{
			Name:        "color",
			Usage:       "use color in display output",
			DefaultText: "depends on output being a TTY",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.IsSet("color") {
			color.NoColor = !cctx.Bool("color")
		}

		act := cctx.String("actor")

		conf := config.Conf()
		owner, worker, _, _, controlAddresses, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("name"),
			tablewriter.Col("ID"),
			tablewriter.Col("key"),
			tablewriter.Col("balance"),
		)

		printKey := func(name string, a string) {
			b, err := client.LotusWalletBalance(conf.Chain.RpcAddr, conf.Chain.Token, a)
			if err != nil {
				fmt.Printf("%s\t%s: error getting balance: %s\n", name, a, err)
				return
			}

			k, err := client.LotusStateAccountKey(conf.Chain.RpcAddr, conf.Chain.Token, a)
			if err != nil {
				fmt.Printf("%s\t%s: error getting account key: %s\n", name, a, err)
				return
			}

			bstr := types.FIL(b).String()
			switch {
			case b.LessThan(types.FromFil(10)):
				bstr = color.RedString(bstr)
			case b.LessThan(types.FromFil(50)):
				bstr = color.YellowString(bstr)
			default:
				bstr = color.GreenString(bstr)
			}

			tw.Write(map[string]interface{}{
				"name":    name,
				"ID":      a,
				"key":     k,
				"balance": bstr,
			})
		}

		printKey("owner", owner)
		printKey("worker", worker)
		for i, ca := range controlAddresses {
			printKey(fmt.Sprintf("control-%d", i), ca.(string))
		}

		return tw.Flush(os.Stdout)
	},
}

var actorControlSet = &cli.Command{
	Name:      "set",
	Usage:     "Set control address(-es)",
	ArgsUsage: "[...address]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		act := cctx.String("actor")
		maddr, err := address.NewFromString(act)
		if err != nil {
			return fmt.Errorf("parsing address %s: %w", act, err)
		}

		conf := config.Conf()
		ownerStr, workerStr, _, _, controlAddresses, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}
		owner, err := address.NewFromString(ownerStr)
		if err != nil {
			return err
		}
		worker, err := address.NewFromString(workerStr)
		if err != nil {
			return err
		}

		del := map[address.Address]struct{}{}
		existing := map[address.Address]struct{}{}
		for _, controlAddress := range controlAddresses {
			kaStr, err := client.LotusStateAccountKey(conf.Chain.RpcAddr, conf.Chain.Token, controlAddress.(string))
			if err != nil {
				return err
			}

			ka, err := address.NewFromString(kaStr)
			if err != nil {
				return err
			}
			log.Infow("LotusStateAccountKey", "controlAddress", controlAddress, "ka", ka)

			del[ka] = struct{}{}
			existing[ka] = struct{}{}
		}

		var toSet []address.Address

		for _, as := range cctx.Args().Slice() {
			kaStr, err := client.LotusStateAccountKey(conf.Chain.RpcAddr, conf.Chain.Token, as)
			if err != nil {
				return err
			}

			ka, err := address.NewFromString(kaStr)
			if err != nil {
				return err
			}

			// make sure the address exists on chain
			_, err = client.LotusStateLookupID(conf.Chain.RpcAddr, conf.Chain.Token, ka.String())
			if err != nil {
				return xerrors.Errorf("looking up %s: %w", ka, err)
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

		msgCid, err := send(nk, &types.Message{
			From:   owner,
			To:     maddr,
			Method: builtin.MethodsMiner.ChangeWorkerAddress,
			Value:  big.Zero(),
			Params: sp,
		})
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Printf("Message CID: %s\n", msgCid.String())

		return waitMsg(msgCid.String())
	},
}

var actorProposeChangeWorker = &cli.Command{
	Name:      "propose-change-worker",
	Usage:     "Propose a worker address change",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of new worker address")
		}

		if !cctx.Bool("really-do-it") {
			fmt.Fprintln(cctx.App.Writer, "Pass --really-do-it to actually execute this action")
			return nil
		}

		act := cctx.String("actor")
		maddr, err := address.NewFromString(act)
		if err != nil {
			return fmt.Errorf("parsing address %s: %w", act, err)
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

		ownerStr, workerStr, newWorkerStr, _, controlAddressesStr, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		owner, err := address.NewFromString(ownerStr)
		if err != nil {
			return err
		}

		if newWorkerStr == emptyWorker {
			if workerStr == newAddr.String() {
				return fmt.Errorf("worker address already set to %s", na)
			}
		} else {
			if newWorkerStr == newAddr.String() {
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

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		nk, err := getAccount(cctx)
		if err != nil {
			return err
		}

		msgCid, err := send(nk, &types.Message{
			From:   owner,
			To:     maddr,
			Method: builtin.MethodsMiner.ChangeWorkerAddress,
			Value:  big.Zero(),
			Params: sp,
		})
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Printf("Message CID: %s\n", msgCid.String())

		err = waitMsg(msgCid.String())
		if err != nil {
			return err
		}

		_, _, _, workerChangeEpoch, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		fmt.Fprintf(cctx.App.Writer, "Worker key change to %s successfully proposed.\n", na)
		fmt.Fprintf(cctx.App.Writer, "Call 'confirm-change-worker' at or after height %d to complete.\n", workerChangeEpoch)

		return nil
	},
}

var actorConfirmChangeWorker = &cli.Command{
	Name:      "confirm-change-worker",
	Usage:     "Confirm a worker address change",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "actor",
			Usage:    "specify the address of miner actor",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of new worker address")
		}

		if !cctx.Bool("really-do-it") {
			fmt.Fprintln(cctx.App.Writer, "Pass --really-do-it to actually execute this action")
			return nil
		}

		act := cctx.String("actor")
		maddr, err := address.NewFromString(act)
		if err != nil {
			return fmt.Errorf("parsing address %s: %w", act, err)
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

		ownerStr, _, newWorkerStr, workerChangeEpoch, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		owner, err := address.NewFromString(ownerStr)
		if err != nil {
			return err
		}

		if newWorkerStr == emptyWorker {
			return xerrors.Errorf("no worker key change proposed")
		} else if newWorkerStr != newAddr.String() {
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

		msgCid, err := send(nk, &types.Message{
			From:   owner,
			To:     maddr,
			Method: builtin.MethodsMiner.ConfirmUpdateWorkerKey,
			Value:  big.Zero(),
		})
		if err != nil {
			log.Error(err)
			return err
		}

		fmt.Printf("Confirm Message CID: %s\n", msgCid.String())

		err = waitMsg(msgCid.String())
		if err != nil {
			return err
		}

		_, workerStr, _, _, _, err := client.LotusStateMinerInfo(conf.Chain.RpcAddr, conf.Chain.Token, act)
		if err != nil {
			return err
		}

		if workerStr != newAddr.String() {
			return fmt.Errorf("Confirmed worker address change not reflected on chain: expected '%s', found '%s'", newAddr, workerStr)
		}

		return nil
	},
}
