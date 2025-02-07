package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.sia.tech/renterd/api"
	"go.sia.tech/renterd/webhooks"
)

const (
	cacheKeyDownloadContracts = "downloadcontracts"
	cacheKeyGougingParams     = "gougingparams"

	cacheEntryExpiry = 5 * time.Minute
)

var (
	errCacheNotReady = errors.New("cache is not ready yet, required webhooks have not been registered")
	errCacheOutdated = errors.New("cache is outdated, the value fetched from the bus does not match the cached value")
)

type memoryCache struct {
	items map[string]*cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	value  interface{}
	expiry time.Time
}

func newMemoryCache() *memoryCache {
	return &memoryCache{
		items: make(map[string]*cacheEntry),
	}
}

func (c *memoryCache) Get(key string) (value interface{}, found bool, expired bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok {
		return nil, false, false
	} else if time.Now().After(entry.expiry) {
		return entry.value, true, true
	}
	return entry.value, true, false
}

func (c *memoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &cacheEntry{
		value:  value,
		expiry: time.Now().Add(cacheEntryExpiry),
	}
}

func (c *memoryCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

type (
	Bus interface {
		Contracts(ctx context.Context, opts api.ContractsOpts) ([]api.ContractMetadata, error)
		GougingParams(ctx context.Context) (api.GougingParams, error)
		RegisterWebhook(ctx context.Context, wh webhooks.Webhook, opts ...webhooks.HeaderOption) error
	}

	WorkerCache interface {
		DownloadContracts(ctx context.Context) ([]api.ContractMetadata, error)
		GougingParams(ctx context.Context) (api.GougingParams, error)
		HandleEvent(event webhooks.Event) error
		Initialize(ctx context.Context, workerAPI string, opts ...webhooks.HeaderOption) error
	}
)

type cache struct {
	b Bus

	cache  *memoryCache
	logger *zap.SugaredLogger

	mu    sync.Mutex
	ready bool
}

func NewCache(b Bus, logger *zap.Logger) WorkerCache {
	return &cache{
		b: b,

		cache:  newMemoryCache(),
		logger: logger.Sugar().Named("workercache"),
	}
}

func (c *cache) DownloadContracts(ctx context.Context) (contracts []api.ContractMetadata, err error) {
	// fetch directly from bus if the cache is not ready
	if !c.isReady() {
		c.logger.Warn(errCacheNotReady)
		contracts, err = c.b.Contracts(ctx, api.ContractsOpts{})
		return
	}

	// fetch from bus if it's not cached or expired
	value, found, expired := c.cache.Get(cacheKeyDownloadContracts)
	if !found || expired {
		contracts, err = c.b.Contracts(ctx, api.ContractsOpts{})
		if err == nil {
			c.cache.Set(cacheKeyDownloadContracts, contracts)
		}
		if expired && !contractsEqual(value.([]api.ContractMetadata), contracts) {
			c.logger.Warn(fmt.Errorf("%w: key %v", errCacheOutdated, cacheKeyDownloadContracts))
		}
		return
	}

	return value.([]api.ContractMetadata), nil
}

func (c *cache) GougingParams(ctx context.Context) (gp api.GougingParams, err error) {
	// fetch directly from bus if the cache is not ready
	if !c.isReady() {
		c.logger.Warn(errCacheNotReady)
		gp, err = c.b.GougingParams(ctx)
		return
	}

	// fetch from bus if it's not cached or expired
	value, found, expired := c.cache.Get(cacheKeyGougingParams)
	if !found || expired {
		gp, err = c.b.GougingParams(ctx)
		if err == nil {
			c.cache.Set(cacheKeyGougingParams, gp)
		}
		if expired && !gougingParamsEqual(value.(api.GougingParams), gp) {
			c.logger.Warn(fmt.Errorf("%w: key %v", errCacheOutdated, cacheKeyGougingParams))
		}
		return
	}

	return value.(api.GougingParams), nil
}

func (c *cache) HandleEvent(event webhooks.Event) (err error) {
	log := c.logger.With("module", event.Module, "event", event.Event)

	// parse the event
	parsed, err := api.ParseEventWebhook(event)
	if err != nil {
		log.Errorw("failed to parse event", "error", err)
		return err
	}

	// handle the event
	switch e := parsed.(type) {
	case api.EventConsensusUpdate:
		log = log.With("bh", e.BlockHeight, "ts", e.Timestamp)
		c.handleConsensusUpdate(e)
	case api.EventContractArchive:
		log = log.With("fcid", e.ContractID, "ts", e.Timestamp)
		c.handleContractArchive(e)
	case api.EventContractRenew:
		log = log.With("fcid", e.Renewal.ID, "renewedFrom", e.Renewal.RenewedFrom, "ts", e.Timestamp)
		c.handleContractRenew(e)
	case api.EventSettingUpdate:
		log = log.With("key", e.Key, "ts", e.Timestamp)
		err = c.handleSettingUpdate(e)
	case api.EventSettingDelete:
		log = log.With("key", e.Key, "ts", e.Timestamp)
		c.handleSettingDelete(e)
	default:
		log.Info("unhandled event", e)
		return
	}

	// log the outcome
	if err != nil {
		log.Errorw("failed to handle event", "error", err)
	} else {
		log.Info("handled event")
	}
	return
}

func (c *cache) Initialize(ctx context.Context, workerAPI string, webhookOpts ...webhooks.HeaderOption) error {
	eventsURL := fmt.Sprintf("%s/events", workerAPI)
	for _, wh := range []webhooks.Webhook{
		webhooks.NewEventWebhook(eventsURL, api.EventConsensusUpdate{}),
		webhooks.NewEventWebhook(eventsURL, api.EventContractArchive{}),
		webhooks.NewEventWebhook(eventsURL, api.EventContractRenew{}),
		webhooks.NewEventWebhook(eventsURL, api.EventSettingUpdate{}),
	} {
		if err := c.b.RegisterWebhook(ctx, wh, webhookOpts...); err != nil {
			return fmt.Errorf("failed to register webhook '%s', err: %v", wh, err)
		}
	}
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	return nil
}

func (c *cache) isReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready
}

func (c *cache) handleConsensusUpdate(event api.EventConsensusUpdate) {
	// return early if the doesn't have gouging params to update
	value, found, _ := c.cache.Get(cacheKeyGougingParams)
	if !found {
		return
	}

	// update gouging params
	gp := value.(api.GougingParams)
	gp.ConsensusState = event.ConsensusState
	gp.TransactionFee = event.TransactionFee
	c.cache.Set(cacheKeyGougingParams, gp)
}

func (c *cache) handleContractArchive(event api.EventContractArchive) {
	// return early if the cache doesn't have contracts
	value, found, _ := c.cache.Get(cacheKeyDownloadContracts)
	if !found {
		return
	}
	contracts := value.([]api.ContractMetadata)

	// remove the contract from the cache
	for i, contract := range contracts {
		if contract.ID == event.ContractID {
			contracts = append(contracts[:i], contracts[i+1:]...)
			break
		}
	}
	c.cache.Set(cacheKeyDownloadContracts, contracts)
}

func (c *cache) handleContractRenew(event api.EventContractRenew) {
	// return early if the cache doesn't have contracts
	value, found, _ := c.cache.Get(cacheKeyDownloadContracts)
	if !found {
		return
	}
	contracts := value.([]api.ContractMetadata)

	// update the renewed contract in the cache
	for i, contract := range contracts {
		if contract.ID == event.Renewal.RenewedFrom {
			contracts[i] = event.Renewal
			break
		}
	}

	c.cache.Set(cacheKeyDownloadContracts, contracts)
}

func (c *cache) handleSettingDelete(e api.EventSettingDelete) {
	if e.Key == api.SettingGouging || e.Key == api.SettingRedundancy {
		c.cache.Invalidate(cacheKeyGougingParams)
	}
}

func (c *cache) handleSettingUpdate(e api.EventSettingUpdate) (err error) {
	// return early if the cache doesn't have gouging params to update
	value, found, _ := c.cache.Get(cacheKeyGougingParams)
	if !found {
		return nil
	}
	gp := value.(api.GougingParams)

	// marshal the updated value
	data, err := json.Marshal(e.Update)
	if err != nil {
		return fmt.Errorf("couldn't marshal the given value, error: %v", err)
	}

	// unmarshal into the appropriated setting and update the cache
	switch e.Key {
	case api.SettingGouging:
		var gs api.GougingSettings
		if err := json.Unmarshal(data, &gs); err != nil {
			return fmt.Errorf("couldn't update gouging settings, invalid request body, %t", e.Update)
		} else if err := gs.Validate(); err != nil {
			return fmt.Errorf("couldn't update gouging settings, error: %v", err)
		}

		gp.GougingSettings = gs
		c.cache.Set(cacheKeyGougingParams, gp)
	case api.SettingRedundancy:
		var rs api.RedundancySettings
		if err := json.Unmarshal(data, &rs); err != nil {
			return fmt.Errorf("couldn't update redundancy settings, invalid request body, %t", e.Update)
		} else if err := rs.Validate(); err != nil {
			return fmt.Errorf("couldn't update redundancy settings, error: %v", err)
		}

		gp.RedundancySettings = rs
		c.cache.Set(cacheKeyGougingParams, gp)
	default:
	}

	return nil
}

func contractsEqual(x, y []api.ContractMetadata) bool {
	if len(x) != len(y) {
		return false
	}
	sort.Slice(x, func(i, j int) bool { return x[i].ID.String() < x[j].ID.String() })
	sort.Slice(y, func(i, j int) bool { return y[i].ID.String() < y[j].ID.String() })
	for i, c := range x {
		if c.ID.String() != y[i].ID.String() {
			return false
		}
	}
	return true
}

func gougingParamsEqual(x, y api.GougingParams) bool {
	var xb bytes.Buffer
	var yb bytes.Buffer
	json.NewEncoder(&xb).Encode(x)
	json.NewEncoder(&yb).Encode(y)
	return bytes.Equal(xb.Bytes(), yb.Bytes())
}
