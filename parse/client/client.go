package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/desmos-labs/juno/config"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// ClientProxy implements a wrapper around both a Tendermint RPC client and a
// Cosmos Sdk REST client that allows for essential data queries.
type ClientProxy struct {
	rpcClient  rpcclient.Client // Tendermint RPC node
	clientNode string           // Full node
	cdc        *codec.Codec
}

func New(cfg config.Config, codec *codec.Codec) (ClientProxy, error) {
	rpcClient, err := rpcclient.NewHTTP(cfg.RPCNode, "/websocket")
	if err != nil {
		return ClientProxy{}, err
	}

	if err := rpcClient.Start(); err != nil {
		return ClientProxy{}, err
	}

	return ClientProxy{rpcClient: rpcClient, clientNode: cfg.ClientNode, cdc: codec}, nil
}

// LatestHeight returns the latest block height on the active chain. An error
// is returned if the query fails.
func (cp ClientProxy) LatestHeight() (int64, error) {
	status, err := cp.rpcClient.Status()
	if err != nil {
		return -1, err
	}

	height := status.SyncInfo.LatestBlockHeight
	return height, nil
}

// Block queries for a block by height. An error is returned if the query fails.
func (cp ClientProxy) Block(height int64) (*tmctypes.ResultBlock, error) {
	return cp.rpcClient.Block(&height)
}

func (cp ClientProxy) BlockResults(height int64) (*tmctypes.ResultBlockResults, error) {
	return cp.rpcClient.BlockResults(&height)
}

// TendermintTx queries for a transaction by hash. An error is returned if the
// query fails.
func (cp ClientProxy) TendermintTx(hash string) (*tmctypes.ResultTx, error) {
	hashRaw, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}

	return cp.rpcClient.Tx(hashRaw, false)
}

// Validators returns all the known Tendermint validators for a given block
// height. An error is returned if the query fails.
func (cp ClientProxy) Validators(height int64) (*tmctypes.ResultValidators, error) {
	return cp.rpcClient.Validators(&height, 0, 1000000)
}

// Genesis returns the genesis state
func (cp ClientProxy) Genesis() (*tmctypes.ResultGenesis, error) {
	return cp.rpcClient.Genesis()
}

// Stop defers the node stop execution to the RPC client.
func (cp ClientProxy) Stop() error {
	return cp.rpcClient.Stop()
}

// SubscribeNewBlocks subscribes to the new block event handler through the RPC
// client with the given subscriber name. An receiving only channel, context
// cancel function and an error is returned. It is up to the caller to cancel
// the context and handle any errors appropriately.
func (cp ClientProxy) SubscribeNewBlocks(subscriber string) (<-chan tmctypes.ResultEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	eventCh, err := cp.rpcClient.Subscribe(ctx, subscriber, "tm.event = 'NewBlock'")
	return eventCh, cancel, err
}

// SubscribeEvents subscribes to the new event handler based on the given query
// through the RPC  client with the given subscriber name. An receiving only channel,
// context  cancel function and an error is returned. It is up to the caller to cancel
// the context and handle any errors appropriately.
func (cp ClientProxy) SubscribeEvents(subscriber string, query string) (<-chan tmctypes.ResultEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	eventCh, err := cp.rpcClient.Subscribe(ctx, subscriber, query)
	return eventCh, cancel, err
}

// QueryLCD queries the LCD at the given endpoint, and deserializes the result into the given pointer.
// If an error is raised, retuns the error
func (cp ClientProxy) QueryLCD(endpoint string, ptr interface{}) error {
	resp, err := http.Get(fmt.Sprintf("%s/%s", cp.clientNode, endpoint))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := cp.cdc.UnmarshalJSON(bz, ptr); err != nil {
		return err
	}

	return nil
}

// Tx queries for a transaction from the REST client and decodes it into a sdk.Tx
// if the transaction exists. An error is returned if the tx doesn't exist or
// decoding fails.
func (cp ClientProxy) Tx(hash string) (sdk.TxResponse, error) {
	var tx sdk.TxResponse
	if err := cp.QueryLCD(fmt.Sprintf("txs/%s", hash), &tx); err != nil {
		return sdk.TxResponse{}, err
	}

	return tx, nil
}

// Txs queries for all the transactions in a block. Transactions are returned
// in the sdk.TxResponse format which internally contains an sdk.Tx. An error is
// returned if any query fails.
func (cp ClientProxy) Txs(block *tmctypes.ResultBlock) ([]sdk.TxResponse, error) {
	txResponses := make([]sdk.TxResponse, len(block.Block.Txs), len(block.Block.Txs))

	for i, tmTx := range block.Block.Txs {
		txResponse, err := cp.Tx(fmt.Sprintf("%X", tmTx.Hash()))
		if err != nil {
			return nil, err
		}

		txResponses[i] = txResponse
	}

	return txResponses, nil
}
