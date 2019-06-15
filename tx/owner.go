package tx

import (
	"context"
	"crypto/ecdsa"
	stderr "errors"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

// Callbacks implemented by the WalletOwner
type Callbacks interface {
	// WalletOutOfFunds is called when the wallet owned by the
	// WalletOwner does not have enough funds for a transaction
	WalletOutOfFunds(ctx context.Context, body callback.WalletOutOfFundsBody)
}

// StatusOK defined by ethereum is the value of status
// for a transaction that succeeds
const StatusOK = 1

const gasPrice int64 = 1000000000

var retryConfig = conc.RetryConfig{
	Random:            false,
	UnlimitedAttempts: false,
	Attempts:          2,
	BaseExp:           1,
	BaseTimeout:       time.Second,
	MaxRetryTimeout:   5 * time.Second,
}

type signRequest struct {
	Transaction *types.Transaction
}

type createOwnerRequest struct {
	PrivateKey *ecdsa.PrivateKey
}

// WalletOwner is the only instance that should interact
// with a wallet. Its main goal is to send transactions
// and keep the funding and nonce of the wallet up to
// date
type WalletOwner struct {
	wallet    Wallet
	nonce     uint64
	client    eth.Client
	callbacks Callbacks
	logger    log.Logger
}

type WalletOwnerServices struct {
	Client    eth.Client
	Callbacks Callbacks
	Logger    log.Logger
}

type WalletOwnerProps struct {
	PrivateKey *ecdsa.PrivateKey
	Signer     types.Signer
	Nonce      uint64
}

// NewWalletOwner creates a new instance of a wallet
// owner. The wallet is derived from the private key
// provided
func NewWalletOwner(
	services *WalletOwnerServices,
	props *WalletOwnerProps,
) *WalletOwner {
	wallet := NewWallet(props.PrivateKey, props.Signer)
	executor := &WalletOwner{
		wallet:    wallet,
		nonce:     props.Nonce,
		client:    services.Client,
		callbacks: services.Callbacks,
		logger:    services.Logger.ForClass("tx", "WalletOwner"),
	}

	return executor
}

func (e *WalletOwner) handle(ctx context.Context, ev conc.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case conc.RequestWorkerEvent:
		v, err := e.handleRequestEvent(ctx, ev)
		return v, err
	case conc.ErrorWorkerEvent:
		return e.handleErrorEvent(ctx, ev)
	default:
		panic("received unexpected event type")
	}
}

func (e *WalletOwner) handleRequestEvent(ctx context.Context, ev conc.RequestWorkerEvent) (interface{}, error) {
	switch req := ev.Value.(type) {
	case signRequest:
		return e.signTransaction(req.Transaction)
	case ExecuteRequest:
		return e.executeTransaction(ctx, req)
	default:
		panic("invalid request received for worker")
	}
}

func (e *WalletOwner) handleErrorEvent(ctx context.Context, ev conc.ErrorWorkerEvent) (interface{}, error) {
	// a worker should not be passing errors to the conc.Worker so
	// in that case the error is returned and the execution of the
	// worker should halt
	return nil, ev.Error
}

func (e *WalletOwner) transactionNonce() uint64 {
	nonce := e.nonce
	e.nonce++
	return nonce
}

func (e *WalletOwner) updateNonce(ctx context.Context) errors.Err {
	address := e.wallet.Address().Hex()
	nonce, err := e.client.NonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		err := errors.New(errors.ErrFetchNonce, err)
		e.logger.Debug(ctx, "NonceAt request failed", log.MapFields{
			"call_type": "NonceFailure",
			"address":   address,
		}, err)
		return err
	}

	e.nonce = nonce
	e.logger.Debug(ctx, "", log.MapFields{
		"call_type": "NonceSuccess",
		"address":   address,
		"nonce":     nonce,
	})

	return nil
}

func (e *WalletOwner) signTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	return e.wallet.SignTransaction(tx)
}

func (e *WalletOwner) estimateGas(ctx context.Context, id uint64, address string, data []byte) (uint64, errors.Err) {
	e.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasAttempt",
		"id":        id,
		"address":   address,
	})

	var to *common.Address
	var hex common.Address
	if len(address) > 0 {
		hex = common.HexToAddress(address)
		to = &hex
	}

	gas, err := e.client.EstimateGas(ctx, ethereum.CallMsg{
		From:     e.wallet.Address(),
		To:       to,
		Gas:      0,
		GasPrice: nil,
		Value:    nil,
		Data:     data,
	})

	if err != nil {
		e.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, errors.New(errors.ErrEstimateGas, err)
	}

	// when the gateway fails to estimate the gas of a transaction
	// returns this number which far exceeds the limit of gas in
	// a block. In this case, we should just return an error
	if gas == 2251799813685248 {
		err := stderr.New("gas estimation could not be completed because of execution failure")
		e.logger.Debug(ctx, "", log.MapFields{
			"call_type": "EstimateGasFailure",
			"id":        id,
			"address":   address,
			"err":       err.Error(),
		})
		return 0, errors.New(errors.ErrEstimateGas, err)
	}

	e.logger.Debug(ctx, "", log.MapFields{
		"call_type": "EstimateGasSuccess",
		"id":        id,
		"address":   address,
		"gas":       gas,
	})

	return gas, nil
}

func (e *WalletOwner) generateAndSignTransaction(ctx context.Context, req sendTransactionRequest, gas uint64) (*types.Transaction, error) {
	nonce := e.transactionNonce()

	var tx *types.Transaction
	if len(req.Address) == 0 {
		tx = types.NewContractCreation(nonce,
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	} else {
		tx = types.NewTransaction(nonce, common.HexToAddress(req.Address),
			big.NewInt(0), gas, big.NewInt(gasPrice), req.Data)
	}

	return e.wallet.SignTransaction(tx)
}

type sendTransactionRequest struct {
	ID      uint64
	Address string
	Gas     uint64
	Data    []byte
}

func (e *WalletOwner) sendTransaction(
	ctx context.Context,
	req sendTransactionRequest,
) (eth.SendTransactionResponse, errors.Err) {
	v, err := conc.RetryWithConfig(ctx, conc.SupplierFunc(func() (interface{}, error) {
		tx, err := e.generateAndSignTransaction(ctx, req, req.Gas)
		if err != nil {
			return ExecuteResponse{}, errors.New(errors.ErrSignedTx, err)
		}

		res, err := e.client.SendTransaction(ctx, tx)
		if err != nil {
			switch {
			case err == eth.ErrExceedsBalance:
				e.callbacks.WalletOutOfFunds(ctx, callback.WalletOutOfFundsBody{
					Address: req.Address,
				})

				return eth.SendTransactionResponse{},
					conc.ErrCannotRecover{Cause: errors.New(errors.ErrSendTransaction, err)}

			case err == eth.ErrExceedsBlockLimit:
				return eth.SendTransactionResponse{},
					conc.ErrCannotRecover{Cause: errors.New(errors.ErrSendTransaction, err)}
			case err == eth.ErrInvalidNonce:
				if err := e.updateNonce(ctx); err != nil {
					// if we fail to update the nonce we cannot proceed
					return eth.SendTransactionResponse{},
						conc.ErrCannotRecover{Cause: err}
				}

				return eth.SendTransactionResponse{}, err
			default:
				return eth.SendTransactionResponse{},
					conc.ErrCannotRecover{
						Cause: errors.New(errors.ErrSendTransaction, err),
					}
			}
		}

		return res, nil
	}), retryConfig)

	if err != nil {
		if err, ok := err.(errors.Err); ok {
			return eth.SendTransactionResponse{}, err
		}

		return eth.SendTransactionResponse{}, errors.New(errors.ErrSendTransaction, err)
	}

	return v.(eth.SendTransactionResponse), nil
}

func (e *WalletOwner) executeTransaction(ctx context.Context, req ExecuteRequest) (ExecuteResponse, errors.Err) {
	contractAddress := req.Address
	gas, err := e.estimateGas(ctx, req.ID, req.Address, req.Data)
	if err != nil {
		e.logger.Debug(ctx, "failed to estimate gas", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return ExecuteResponse{}, err
	}

	res, err := e.sendTransaction(ctx, sendTransactionRequest{
		ID:      req.ID,
		Address: req.Address,
		Data:    req.Data,
		Gas:     gas,
	})
	if err != nil {
		return ExecuteResponse{}, err
	}

	if res.Status != StatusOK {
		p, derr := hexutil.Decode(res.Output)
		if derr != nil {
			e.logger.Debug(ctx, "failed to decode the output of the transaction as hex", log.MapFields{
				"call_type": "DecodeTransactionOutputFailure",
				"id":        req.ID,
				"address":   req.Address,
				"err":       derr.Error(),
			})
		}

		output := string(p)
		msg := fmt.Sprintf("transaction receipt has status %d which indicates a transaction execution failure with error %s", res.Status, output)
		err := errors.New(errors.NewErrorCode(errors.InternalError, 1000, msg), stderr.New(msg))
		e.logger.Debug(ctx, "transaction execution failed", log.MapFields{
			"call_type": "ExecuteTransactionFailure",
			"id":        req.ID,
			"address":   req.Address,
		}, err)

		return ExecuteResponse{}, err
	}

	if len(contractAddress) == 0 {
		receipt, err := e.client.TransactionReceipt(ctx, common.HexToHash(res.Hash))
		if err != nil {
			err := errors.New(errors.ErrTransactionReceipt, err)
			e.logger.Debug(ctx, "failure to retrieve transaction receipt", log.MapFields{
				"call_type": "ExecuteTransactionFailure",
				"id":        req.ID,
				"address":   req.Address,
			}, err)

			return ExecuteResponse{}, err
		}

		contractAddress = receipt.ContractAddress.Hex()
	}

	return ExecuteResponse{
		Address: contractAddress,
		Output:  res.Output,
		Hash:    res.Hash,
	}, nil
}