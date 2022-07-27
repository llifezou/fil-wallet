package client

import (
	"bytes"
	"encoding/json"
	"golang.org/x/xerrors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

type Method string

const (
	WalletBalance         Method = "Filecoin.WalletBalance"
	MpoolPush             Method = "Filecoin.MpoolPush"
	GasEstimateMessageGas Method = "Filecoin.GasEstimateMessageGas"
	MpoolGetNonce         Method = "Filecoin.MpoolGetNonce"
)

type client struct {
	rpcAddr string
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

func NewClient(rpcAddr string, method Method, params interface{}) *client {
	rand.Seed(time.Now().UnixNano())
	id := rand.Intn(100)
	return &client{
		rpcAddr: rpcAddr,
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
