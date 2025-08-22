package blockchain

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"go.uber.org/zap"
)

// paymentChannelTimeout is the default upper bound used by local wait helpers
// when waiting for on-chain events related to channel operations.
const paymentChannelTimeout = time.Minute

// MultiPartyEscrowChannel is a lightweight, read-only snapshot of a channel
// in the MultiPartyEscrow contract (sender/recipient/group/value/nonce/expiration/signer).
type MultiPartyEscrowChannel struct {
	Sender     common.Address
	Recipient  common.Address
	GroupID    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
	Signer     common.Address
}

// ChansToWatch groups event channels used by watchers (Open/Extend/AddFunds/Deposit)
// and a shared error channel. Callers should create buffered channels when appropriate.
type ChansToWatch struct {
	ChannelOpens    chan *MultiPartyEscrowChannelOpen
	ChannelExtends  chan *MultiPartyEscrowChannelExtend
	ChannelAddFunds chan *MultiPartyEscrowChannelAddFunds
	DepositFunds    chan *MultiPartyEscrowDepositFunds
	Err             chan error
}

// BindOpts carries pre-built bind.* opts used for calls, txs, filters and watches.
// Contexts embedded into these opts are also used as the operation context.
type BindOpts struct {
	Call     *bind.CallOpts
	Transact *bind.TransactOpts
	Watch    *bind.WatchOpts
	Filter   *bind.FilterOpts
}

// ctxFromBind extracts a non-nil Context from BindOpts (Watch → Call → Transact).
// If none are set, it returns context.TODO() to force explicit context propagation by callers.
func ctxFromBind(opts *BindOpts) context.Context {
	switch {
	case opts != nil && opts.Watch != nil && opts.Watch.Context != nil:
		return opts.Watch.Context
	case opts != nil && opts.Call != nil && opts.Call.Context != nil:
		return opts.Call.Context
	case opts != nil && opts.Transact != nil && opts.Transact.Context != nil:
		return opts.Transact.Context
	default:
		// Avoid Background() here; encourage passing a real, cancellable context.
		return context.TODO()
	}
}

// withTimeout returns ctx if d <= 0, otherwise returns a child context with timeout d.
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// waitOpenID waits for a ChannelOpen event or error/cancellation and returns the opened channel ID.
func waitOpenID(ctx context.Context, ch <-chan *MultiPartyEscrowChannelOpen, errc <-chan error, d time.Duration) (*big.Int, error) {
	c, cancel := withTimeout(ctx, d)
	defer cancel()
	select {
	case ev := <-ch:
		zap.L().Debug("Channel opened", zap.Any("openEvent", ev))
		return ev.ChannelId, nil
	case err := <-errc:
		return nil, fmt.Errorf("watch OpenChannel: %w", err)
	case <-c.Done():
		return nil, c.Err()
	}
}

// waitExtendID waits for a ChannelExtend event or error/cancellation and returns the channel ID.
func waitExtendID(ctx context.Context, ch <-chan *MultiPartyEscrowChannelExtend, errc <-chan error, d time.Duration) (*big.Int, error) {
	c, cancel := withTimeout(ctx, d)
	defer cancel()
	select {
	case ev := <-ch:
		zap.L().Debug("Channel extended", zap.Any("extendEvent", ev))
		return ev.ChannelId, nil
	case err := <-errc:
		return nil, fmt.Errorf("watch ChannelExtend: %w", err)
	case <-c.Done():
		return nil, c.Err()
	}
}

// waitAddFundsID waits for a ChannelAddFunds event or error/cancellation and returns the channel ID.
func waitAddFundsID(ctx context.Context, ch <-chan *MultiPartyEscrowChannelAddFunds, errc <-chan error, d time.Duration) (*big.Int, error) {
	c, cancel := withTimeout(ctx, d)
	defer cancel()
	select {
	case ev := <-ch:
		zap.L().Debug("Channel funds added", zap.Any("addFundsEvent", ev))
		return ev.ChannelId, nil
	case err := <-errc:
		return nil, fmt.Errorf("watch ChannelAddFunds: %w", err)
	case <-c.Done():
		return nil, c.Err()
	}
}

// waitDeposit waits for a DepositFunds event or error/cancellation and returns when observed.
func waitDeposit(ctx context.Context, ch <-chan *MultiPartyEscrowDepositFunds, errc <-chan error, d time.Duration) error {
	c, cancel := withTimeout(ctx, d)
	defer cancel()
	select {
	case ev := <-ch:
		zap.L().Debug("Deposited to MPE", zap.Any("depositFundsEvent", ev))
		return nil
	case err := <-errc:
		return fmt.Errorf("watch DepositFunds: %w", err)
	case <-c.Done():
		return c.Err()
	}
}

// FilterChannels scans ChannelOpen events with given filters and returns the latest matching event
// for (sender == signer == senders[0], recipient == recipients[0], groupID == groupIDs[0]) if any.
// The iterator is closed regardless of path; caller owns filterOpts lifecycle.
func (eth *EVMClient) FilterChannels(senders, recipients []common.Address, groupIDs [][32]byte, filterOpts *bind.FilterOpts) (*MultiPartyEscrowChannelOpen, error) {
	it, err := eth.MPE.FilterChannelOpen(filterOpts, senders, recipients, groupIDs)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := it.Close(); cerr != nil {
			zap.L().Error("error closing channel open iterator", zap.Error(cerr))
		}
	}()

	var filtered *MultiPartyEscrowChannelOpen
	for it.Next() {
		ev := it.Event
		if ev.Sender == senders[0] && ev.Signer == senders[0] && ev.Recipient == recipients[0] && ev.GroupId == groupIDs[0] {
			zap.L().Debug("Filtered eventChannelOpen", zap.Any("eventChannelOpen", ev))
			filtered = ev
		}
	}
	if err = it.Error(); err != nil {
		return nil, err
	}
	return filtered, nil
}

// DecodePaymentGroupID decodes a base64-encoded payment group ID into a [32]byte.
func DecodePaymentGroupID(encoded string) ([32]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return [32]byte{}, err
	}
	var groupID [32]byte
	copy(groupID[:], decoded)
	return groupID, nil
}

// GetCallOpts builds bind.CallOpts with a fixed block number and context.
func GetCallOpts(fromAddress common.Address, currentBlockNumber *big.Int, ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{Pending: false, From: fromAddress, BlockNumber: currentBlockNumber, Context: ctx}
}

// GetWatchOpts builds bind.WatchOpts starting from currentBlockNumber and using ctx.
func GetWatchOpts(currentBlockNumber *big.Int, ctx context.Context) *bind.WatchOpts {
	start := currentBlockNumber.Uint64()
	return &bind.WatchOpts{Start: &start, Context: ctx}
}

// GetFilterOpts builds bind.FilterOpts from genesis (Start=0) to currentBlockNumber and using ctx.
func GetFilterOpts(currentBlockNumber *big.Int, ctx context.Context) *bind.FilterOpts {
	end := currentBlockNumber.Uint64()
	return &bind.FilterOpts{Start: 0, End: &end, Context: ctx}
}

// GetTransactOpts creates a transactor bound to the given chainID and ECDSA key.
func GetTransactOpts(chainID *big.Int, pk *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		zap.L().Error("failed to create transactor", zap.Error(err))
		return nil, err
	}
	return opts, nil
}

// GetNewExpiration returns a new expiration block = current + threshold + small offset.
// The extra offset (240 blocks) gives a buffer to avoid near-expiry channels.
func GetNewExpiration(currentBlockNumber, threshold *big.Int) *big.Int {
	// default + small offset blocks
	return new(big.Int).Add(new(big.Int).Add(currentBlockNumber, threshold), big.NewInt(240))
}

// watchChannelOpen subscribes to ChannelOpen events with a context-aware WatchOpts.
// If errc is non-nil, subscription errors are forwarded there.
func (eth *EVMClient) watchChannelOpen(ctx context.Context, watch *bind.WatchOpts, out chan *MultiPartyEscrowChannelOpen, errc chan error, senders, recipients []common.Address, groupIDs [][32]byte) (event.Subscription, error) {
	w := *watch
	w.Context = ctx
	sub, err := eth.MPE.WatchChannelOpen(&w, out, senders, recipients, groupIDs)
	if err != nil {
		if errc != nil {
			errc <- err
		}
		return nil, err
	}
	if errc != nil {
		go func() {
			select {
			case e := <-sub.Err():
				if e != nil {
					errc <- e
				}
			case <-ctx.Done():
			}
		}()
	}
	return sub, nil
}

// watchDepositFunds subscribes to DepositFunds events and forwards errors to errc if provided.
func (eth *EVMClient) watchDepositFunds(ctx context.Context, watch *bind.WatchOpts, out chan *MultiPartyEscrowDepositFunds, errc chan error, senders []common.Address) (event.Subscription, error) {
	w := *watch
	w.Context = ctx
	sub, err := eth.MPE.WatchDepositFunds(&w, out, senders)
	if err != nil {
		if errc != nil {
			errc <- err
		}
		return nil, err
	}
	if errc != nil {
		go func() {
			select {
			case e := <-sub.Err():
				if e != nil {
					errc <- e
				}
			case <-ctx.Done():
			}
		}()
	}
	return sub, nil
}

// watchChannelAddFunds subscribes to ChannelAddFunds events and forwards errors to errc if provided.
func (eth *EVMClient) watchChannelAddFunds(ctx context.Context, watch *bind.WatchOpts, out chan *MultiPartyEscrowChannelAddFunds, errc chan error, channelIDs []*big.Int) (event.Subscription, error) {
	w := *watch
	w.Context = ctx
	sub, err := eth.MPE.WatchChannelAddFunds(&w, out, channelIDs)
	if err != nil {
		if errc != nil {
			errc <- err
		}
		return nil, err
	}
	if errc != nil {
		go func() {
			select {
			case e := <-sub.Err():
				if e != nil {
					errc <- e
				}
			case <-ctx.Done():
			}
		}()
	}
	return sub, nil
}

// watchChannelExtend subscribes to ChannelExtend events and forwards errors to errc if provided.
func (eth *EVMClient) watchChannelExtend(ctx context.Context, watch *bind.WatchOpts, out chan *MultiPartyEscrowChannelExtend, errc chan error, channelIDs []*big.Int) (event.Subscription, error) {
	w := *watch
	w.Context = ctx
	sub, err := eth.MPE.WatchChannelExtend(&w, out, channelIDs)
	if err != nil {
		if errc != nil {
			errc <- err
		}
		return nil, err
	}
	if errc != nil {
		go func() {
			select {
			case e := <-sub.Err():
				if e != nil {
					errc <- e
				}
			case <-ctx.Done():
			}
		}()
	}
	return sub, nil
}

// getChannelStateFromBlockchain returns a channel snapshot from MPE.Channels.
// It returns (nil,false,err) on read error, (nil,false,err) if the record looks empty,
// or (channel,true,nil) on success.
func (eth *EVMClient) getChannelStateFromBlockchain(channelID *big.Int) (*MultiPartyEscrowChannel, bool, error) {
	ch, err := eth.MPE.Channels(nil, channelID)
	if err != nil {
		return nil, false, err
	}
	var zero common.Address
	if ch.Sender == zero {
		return nil, false, errors.New("incorrect sender of channel")
	}
	channel := &MultiPartyEscrowChannel{
		Sender:     ch.Sender,
		Recipient:  ch.Recipient,
		GroupID:    ch.GroupId,
		Value:      ch.Value,
		Nonce:      ch.Nonce,
		Expiration: ch.Expiration,
		Signer:     ch.Signer,
	}
	zap.L().Debug("Channel state from blockchain", zap.Any("channel", channel))
	return channel, true, nil
}

// GetCurrentBlockNumberCtx returns the latest block number using the provided context.
func (eth *EVMClient) GetCurrentBlockNumberCtx(ctx context.Context) (*big.Int, error) {
	header, err := eth.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		zap.L().Error("failed to get last block number", zap.Error(err))
		return nil, err
	}
	return header.Number, nil
}

// WaitForTransaction polls for a transaction receipt with exponential backoff,
// until receipt is available, context is done, or an error occurs. If maxBackoff
// is non-zero, backoff will not exceed it. It returns an error if the tx is reverted.
func (eth *EVMClient) WaitForTransaction(ctx context.Context, txHash common.Hash, maxBackoff time.Duration) (*types.Receipt, error) {
	backoff := time.Second
	for {
		receipt, err := eth.Client.TransactionReceipt(ctx, txHash)
		switch {
		case err == nil:
			if receipt.Status == types.ReceiptStatusFailed {
				return nil, fmt.Errorf("tx reverted: %s", txHash)
			}
			return receipt, nil
		case errors.Is(err, ethereum.NotFound):
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if maxBackoff == 0 || backoff < maxBackoff {
				backoff *= 2
			}
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil, err
		default:
			return nil, fmt.Errorf("receipt error: %w", err)
		}
	}
}

// GetMPEBalance returns MPE internal balance for callOpts.From.
func (eth *EVMClient) GetMPEBalance(callOpts *bind.CallOpts) (*big.Int, error) {
	bal, err := eth.MPE.Balances(callOpts, callOpts.From)
	if err != nil {
		return nil, err
	}
	zap.L().Debug("MPE Balance", zap.Any("mpeBalance", bal))
	return bal, nil
}

// estimateGas returns a shallow copy of tx opts with zero GasLimit for node-side estimation.
func estimateGas(wallet *bind.TransactOpts) *bind.TransactOpts {
	return &bind.TransactOpts{From: wallet.From, Signer: wallet.Signer, Value: nil, GasLimit: 0}
}

// ensureAllowance checks token allowance(owner→spender) and, if insufficient,
// submits Approve(max) and waits for the tx to be mined (with a 30s max backoff).
func (eth *EVMClient) ensureAllowance(ctx context.Context, owner, spender common.Address, need *big.Int, call *bind.CallOpts, txOpts *bind.TransactOpts) error {
	allowance, err := eth.FetchToken.Allowance(call, owner, spender)
	if err != nil {
		return err
	}
	if allowance != nil && allowance.Cmp(need) >= 0 {
		return nil
	}
	tx, err := eth.FetchToken.Approve(txOpts, spender, maxUint256)
	if err != nil {
		return err
	}
	_, err = eth.WaitForTransaction(ctx, tx.Hash(), 30*time.Second)
	return err
}

// availableAmount returns (on-chain channel value - currently signed amount).
func availableAmount(onchainValue, currentSigned *big.Int) *big.Int {
	return new(big.Int).Sub(onchainValue, currentSigned)
}

// EnsurePaymentChannel guarantees there is a valid channel (sufficient funds and expiration)
// for (sender, recipient, groupID). It may deposit/open/extend/addFunds as needed,
// waiting for corresponding events. Returns the channel ID or an error.
func (eth *EVMClient) EnsurePaymentChannel(mpe common.Address, filtered *MultiPartyEscrowChannelOpen, currentSigned, price, desiredExpiration *big.Int, opts *BindOpts, chans *ChansToWatch, senders, recipients []common.Address, groupIDs [][32]byte) (*big.Int, error) {
	// Use a single base context for all operations below.
	baseCtx := ctxFromBind(opts)

	var err error
	filtered, err = eth.FilterChannels(senders, recipients, groupIDs, opts.Filter)
	if err != nil {
		return nil, err
	}

	if err = eth.ensureAllowance(baseCtx, senders[0], mpe, maxUint256, opts.Call, opts.Transact); err != nil {
		return nil, err
	}

	if filtered == nil {
		return eth.OpenNewChannel(price, desiredExpiration, opts, chans, senders, recipients, groupIDs)
	}
	return eth.EnsureChannelValidity(filtered, currentSigned, price, desiredExpiration, opts, chans)
}

// OpenNewChannel opens a new MPE channel. If the MPE internal balance is insufficient,
// it performs DepositAndOpenChannel and waits for both DepositFunds and ChannelOpen events.
func (eth *EVMClient) OpenNewChannel(price, desiredExpiration *big.Int, opts *BindOpts, chans *ChansToWatch, senders, recipients []common.Address, groupIDs [][32]byte) (*big.Int, error) {
	ctx := ctxFromBind(opts)

	mpeBal, err := eth.MPE.Balances(opts.Call, senders[0])
	if err != nil {
		return nil, err
	}

	openDirect := func() (*big.Int, error) {
		sub, err := eth.watchChannelOpen(ctx, opts.Watch, chans.ChannelOpens, chans.Err, senders, recipients, groupIDs)
		if err != nil {
			return nil, err
		}
		defer sub.Unsubscribe()

		if _, err = eth.MPE.OpenChannel(estimateGas(opts.Transact), senders[0], recipients[0], groupIDs[0], price, desiredExpiration); err != nil {
			return nil, err
		}
		return waitOpenID(ctx, chans.ChannelOpens, chans.Err, paymentChannelTimeout)
	}

	if mpeBal.Cmp(price) >= 0 {
		return openDirect()
	}

	// Deposit + open: subscribe and wait for both events in parallel.
	subOpen, err := eth.watchChannelOpen(ctx, opts.Watch, chans.ChannelOpens, chans.Err, senders, recipients, groupIDs)
	if err != nil {
		return nil, err
	}
	defer subOpen.Unsubscribe()

	subDep, err := eth.watchDepositFunds(ctx, opts.Watch, chans.DepositFunds, chans.Err, senders)
	if err != nil {
		return nil, err
	}
	defer subDep.Unsubscribe()

	if _, err = eth.MPE.DepositAndOpenChannel(estimateGas(opts.Transact), senders[0], recipients[0], groupIDs[0], price, desiredExpiration); err != nil {
		return nil, err
	}

	var (
		idMu sync.Mutex
		id   *big.Int
		wg   sync.WaitGroup
		wErr = make(chan error, 2)
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		got, err := waitOpenID(ctx, chans.ChannelOpens, chans.Err, paymentChannelTimeout)
		if err != nil {
			wErr <- err
			return
		}
		idMu.Lock()
		id = got
		idMu.Unlock()
	}()
	go func() {
		defer wg.Done()
		wErr <- waitDeposit(ctx, chans.DepositFunds, chans.Err, paymentChannelTimeout)
	}()

	wg.Wait()
	close(wErr)
	for e := range wErr {
		if e != nil {
			return nil, e
		}
	}
	if id == nil {
		return nil, errors.New("channel open timeout")
	}
	return id, nil
}

// EnsureChannelValidity ensures an opened channel has enough funds and a long-enough expiration.
// It may deposit to MPE, AddFunds, Extend, or ExtendAndAddFunds and waits for the corresponding events.
func (eth *EVMClient) EnsureChannelValidity(opened *MultiPartyEscrowChannelOpen, currentSigned, price, newExpiration *big.Int, opts *BindOpts, chans *ChansToWatch) (*big.Int, error) {
	ctx := ctxFromBind(opts)

	avail := availableAmount(opened.Amount, currentSigned)
	needFunds := avail.Cmp(price) < 0
	needExtend := opened.Expiration.Cmp(newExpiration) <= 0
	if !needFunds && !needExtend {
		return opened.ChannelId, nil
	}

	var topUp *big.Int
	if needFunds {
		missing := new(big.Int).Sub(price, avail)
		topUp = missing

		mpeBal, err := eth.MPE.Balances(opts.Call, opened.Sender)
		if err != nil {
			return nil, err
		}
		if mpeBal.Cmp(missing) < 0 {
			subDep, err := eth.watchDepositFunds(ctx, opts.Watch, chans.DepositFunds, chans.Err, []common.Address{opened.Sender})
			if err != nil {
				return nil, err
			}
			defer subDep.Unsubscribe()

			if _, err = eth.MPE.Deposit(estimateGas(opts.Transact), missing); err != nil {
				return nil, err
			}
			if err = waitDeposit(ctx, chans.DepositFunds, chans.Err, paymentChannelTimeout); err != nil {
				return nil, fmt.Errorf("deposit to MPE timeout: %w", err)
			}
		}
	}

	id := opened.ChannelId
	channelIDs := []*big.Int{id}

	switch {
	case needFunds && needExtend:
		subAdd, err := eth.watchChannelAddFunds(ctx, opts.Watch, chans.ChannelAddFunds, chans.Err, channelIDs)
		if err != nil {
			return nil, err
		}
		defer subAdd.Unsubscribe()

		subExt, err := eth.watchChannelExtend(ctx, opts.Watch, chans.ChannelExtends, chans.Err, channelIDs)
		if err != nil {
			return nil, err
		}
		defer subExt.Unsubscribe()

		if _, err = eth.MPE.ChannelExtendAndAddFunds(estimateGas(opts.Transact), id, newExpiration, topUp); err != nil {
			return nil, err
		}

		// Wait for both events in parallel.
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, e := waitExtendID(ctx, chans.ChannelExtends, chans.Err, paymentChannelTimeout)
			errs <- e
		}()
		go func() {
			defer wg.Done()
			_, e := waitAddFundsID(ctx, chans.ChannelAddFunds, chans.Err, paymentChannelTimeout)
			errs <- e
		}()
		wg.Wait()
		close(errs)
		for e := range errs {
			if e != nil {
				return nil, e
			}
		}

	case needFunds:
		subAdd, err := eth.watchChannelAddFunds(ctx, opts.Watch, chans.ChannelAddFunds, chans.Err, channelIDs)
		if err != nil {
			return nil, err
		}
		defer subAdd.Unsubscribe()

		if _, err = eth.MPE.ChannelAddFunds(estimateGas(opts.Transact), id, topUp); err != nil {
			return nil, err
		}
		if _, err = waitAddFundsID(ctx, chans.ChannelAddFunds, chans.Err, paymentChannelTimeout); err != nil {
			return nil, err
		}

	default:
		subExt, err := eth.watchChannelExtend(ctx, opts.Watch, chans.ChannelExtends, chans.Err, channelIDs)
		if err != nil {
			return nil, err
		}
		defer subExt.Unsubscribe()

		if _, err = eth.MPE.ChannelExtend(estimateGas(opts.Transact), id, newExpiration); err != nil {
			return nil, err
		}
		if _, err = waitExtendID(ctx, chans.ChannelExtends, chans.Err, paymentChannelTimeout); err != nil {
			return nil, err
		}
	}

	return id, nil
}
