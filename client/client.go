package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	lotusClient "github.com/filecoin-project/lotus/api/client"
	"golang.org/x/xerrors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Method string

const (
	ChainHead                  Method = "Filecoin.ChainHead"
	WalletBalance              Method = "Filecoin.WalletBalance"
	MpoolPush                  Method = "Filecoin.MpoolPush"
	GasEstimateMessageGas      Method = "Filecoin.GasEstimateMessageGas"
	MpoolGetNonce              Method = "Filecoin.MpoolGetNonce"
	StateLookupID              Method = "Filecoin.StateLookupID"
	StateGetActor              Method = "Filecoin.StateGetActor"
	StateMinerInfo             Method = "Filecoin.StateMinerInfo"
	StateWaitMsgLimited        Method = "Filecoin.StateWaitMsgLimited"
	StateSearchMsg             Method = "Filecoin.StateSearchMsg"
	StateMinerAvailableBalance Method = "Filecoin.StateMinerAvailableBalance"
	StateAccountKey            Method = "Filecoin.StateAccountKey"
)

type client struct {
	rpcAddr string
	token   string
	JsonRpc string      `json:"jsonrpc"` // "2.0"
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int         `json:"id"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Id      int         `json:"id"`
	Error   interface{} `json:"error"`
}

func NewClient(rpcAddr, token string, method Method, params interface{}) *client {
	rand.Seed(time.Now().UnixNano())
	id := rand.Intn(100)
	return &client{
		rpcAddr: rpcAddr,
		token:   token,
		JsonRpc: "2.0",
		Method:  string(method),
		Params:  params,
		Id:      id,
	}
}

func (c *client) Call() ([]byte, error) {
	dataByte, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.rpcAddr, bytes.NewBuffer(dataByte))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if len(strings.Trim(c.token, " ")) > 0 {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, xerrors.New(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func NewLotusAPI(rpcAddr, token string) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestHeader := http.Header{}
	requestHeader.Add("Content-Type", "application/json")

	if token != "" {
		tokenHeader := fmt.Sprintf("Bearer %s", token)
		requestHeader.Set("Authorization", tokenHeader)
	}

	lotusAPI, closer, err := lotusClient.NewFullNodeRPCV1(context.Background(), rpcAddr, requestHeader)
	return lotusAPI, closer, err
}
