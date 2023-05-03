package client

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"testing"
)

func TestLotusGasEstimateMessageGas(t *testing.T) {
	to, _ := address.NewFromString("f1b2j6uc4mxxd5yqw2d7jgae4wsf3knvlwtuhinpy")
	from, _ := address.NewFromString("f3q5shp4pirwajvhpm3rlnho3t3l4bnejaba3vnewckcvbkwzr62rklof7yismmv4ktvb4zfoucewcnktxntba")
	gasLimit, gasFeeCap, gasPremium, err := LotusGasEstimateMessageGas("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", &types.Message{
		To:     to,
		From:   from,
		Nonce:  1,
		Method: 0,
		Value:  abi.NewTokenAmount(1000000000000000000),
	}, 1000000000000000000)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(gasLimit, gasFeeCap, gasPremium)
}

func TestLotusMpoolGetNonce(t *testing.T) {
	nonce, err := LotusMpoolGetNonce("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f3q5shp4pirwajvhpm3rlnho3t3l4bnejaba3vnewckcvbkwzr62rklof7yismmv4ktvb4zfoucewcnktxntba")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(nonce)
}

func TestLotusStateLookupID(t *testing.T) {
	actorID, err := LotusStateLookupID("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f1ys7n5mrm2vtx6coxc5wkmkddan7rznfkax3a6ki")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(actorID)
}
func TestLookupRobustAddress(t *testing.T) {
	actorID, err := LookupRobustAddress("https://api.node.glif.io/rpc/v0", "", "f02098315")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(actorID)
}

func TestLotusStateGetActor(t *testing.T) {
	code, head, nonce, balance, err := LotusStateGetActor("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f2nlvxccdlhydntnt5zchm6uhhpe6og6oy5dsloii")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(code, head, nonce, balance)
}

func TestLotusStateMinerInfo(t *testing.T) {
	owner, worker, newWorker, workerChangeEpoch, controlAddresses, err := LotusStateMinerInfo("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f0688165")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(owner, worker, newWorker, workerChangeEpoch, controlAddresses)
}

func TestLotusStateWaitMsg(t *testing.T) {
	r, err := LotusStateWaitMsgLimited("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "bafy2bzacecl3l5m4a45v6o6ovvmjyhjxmwe7szdmdotphucfbtpvzjtrr744s", 3)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(r)
}

func TestLotusStateSearchMsg(t *testing.T) {
	r, err := LotusStateSearchMsg("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "bafy2bzaceawyq7mhyhr4kdyrnnh5cpuvzam7hujm4pdc2levbmgio3gaf6kuq")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(r)
}

func TestLotusChainHead(t *testing.T) {
	r, err := LotusChainHead("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(r)
}

func TestLotusStateMinerAvailableBalance(t *testing.T) {
	r, err := LotusStateMinerAvailableBalance("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f0154335")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(r)
}

func TestLotusStateAccountKey(t *testing.T) {
	r, err := LotusStateAccountKey("https://26PBnIho1PmwtTUzXfN3pMwS2eK:d3788bb657259d62d1e6ea6acb319073@filecoin.infura.io", "", "f02098315")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(r)
}
