package wallet

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/vm"
	exported7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/exported"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"
	"strconv"
)

var ChainCmd = &cli.Command{
	Name:  "chain",
	Usage: "Interact with filecoin blockchain",
	Subcommands: []*cli.Command{
		decodeCmd,
		encodeCmd,
	},
}

var decodeCmd = &cli.Command{
	Name:  "decode",
	Usage: "decode various types",
	Subcommands: []*cli.Command{
		decodeParamsCmd,
	},
}

var decodeParamsCmd = &cli.Command{
	Name:      "params",
	Usage:     "Decode message params",
	ArgsUsage: "[toAddr method params]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "encoding",
			Value: "base64",
			Usage: "specify input encoding to parse",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 3 {
			return fmt.Errorf("incorrect number of arguments")
		}

		var params []byte
		var err error
		switch cctx.String("encoding") {
		case "base64":
			params, err = base64.StdEncoding.DecodeString(cctx.Args().Get(2))
			if err != nil {
				return xerrors.Errorf("decoding base64 value: %w", err)
			}
		case "hex":
			params, err = hex.DecodeString(cctx.Args().Get(2))
			if err != nil {
				return xerrors.Errorf("decoding hex value: %w", err)
			}
		default:
			return xerrors.Errorf("unrecognized encoding: %s", cctx.String("encoding"))
		}

		method, err := strconv.ParseInt(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return xerrors.Errorf("parsing method id: %w", err)
		}

		decParams, err := decodeParams(abi.MethodNum(method), params)
		if err != nil {
			return err
		}

		fmt.Println(string(decParams))

		return nil
	},
}

var encodeCmd = &cli.Command{
	Name:  "encode",
	Usage: "encode various types",
	Subcommands: []*cli.Command{
		encodeParamsCmd,
	},
}

var encodeParamsCmd = &cli.Command{
	Name:      "params",
	Usage:     "Encodes the given JSON params, encoding: hex",
	ArgsUsage: "[dest method params]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "encoding",
			Value: "base64",
			Usage: "specify input encoding to parse",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 3 {
			return fmt.Errorf("incorrect number of arguments")
		}

		method, err := strconv.ParseInt(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return xerrors.Errorf("parsing method id: %w", err)
		}
		encParams, err := encodeParams(abi.MethodNum(method), cctx.Args().Get(2))
		if err != nil {
			return err
		}

		switch cctx.String("encoding") {
		case "base64", "b64":
			fmt.Println(base64.StdEncoding.EncodeToString(encParams))
		case "hex":
			fmt.Println(hex.EncodeToString(encParams))
		default:
			return xerrors.Errorf("unknown encoding")
		}

		return nil
	},
}

func encodeParams(method abi.MethodNum, params string) ([]byte, error) {
	var paramType cbg.CBORUnmarshaler
	for _, actor := range exported7.BuiltinActors() {
		if MethodMetaMap, ok := filcns.NewActorRegistry().Methods[actor.Code()]; ok {
			var m vm.MethodMeta
			var found bool
			if m, found = MethodMetaMap[abi.MethodNum(method)]; found {
				paramType = reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
			}
		}
	}

	if paramType == nil {
		return nil, fmt.Errorf("unknown method %d", method)
	}

	if err := json.Unmarshal(json.RawMessage(params), &paramType); err != nil {
		return nil, xerrors.Errorf("json unmarshal: %w", err)
	}

	var cbb bytes.Buffer
	if err := paramType.(cbor.Marshaler).MarshalCBOR(&cbb); err != nil {
		return nil, xerrors.Errorf("cbor marshal: %w", err)
	}

	return cbb.Bytes(), nil
}

func decodeParams(method abi.MethodNum, params []byte) ([]byte, error) {
	var paramType cbg.CBORUnmarshaler
	for _, actor := range exported7.BuiltinActors() {
		if MethodMetaMap, ok := filcns.NewActorRegistry().Methods[actor.Code()]; ok {
			var m vm.MethodMeta
			var found bool
			if m, found = MethodMetaMap[abi.MethodNum(method)]; found {
				paramType = reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
			}
		}
	}

	if paramType == nil {
		return nil, fmt.Errorf("unknown method %d", method)
	}

	if err := paramType.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		return nil, err
	}

	return json.MarshalIndent(paramType, "", "  ")
}
