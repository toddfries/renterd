package autopilot

import (
	"net/http"
	"time"

	"go.sia.tech/jape"
	"go.sia.tech/renterd/bus"
	"go.sia.tech/renterd/hostdb"
	"go.sia.tech/renterd/internal/consensus"
	rhpv2 "go.sia.tech/renterd/rhp/v2"
	"go.sia.tech/renterd/wallet"
	"go.sia.tech/renterd/worker"
	"go.sia.tech/siad/types"
)

type Store interface {
	Tip() consensus.ChainIndex
	Synced() bool // TODO: should be added to `Tip` | should be moved over to Bus

	Config() Config
	SetConfig(c Config) error
}

// TODO: should be defined in bus package, should be the interface of the client
type Bus interface {
	// wallet
	WalletBalance() (types.Currency, error)
	WalletAddress() (types.UnlockHash, error)
	WalletTransactions(since time.Time, max int) ([]wallet.Transaction, error)
	WalletFund(txn *types.Transaction, amount types.Currency) ([]types.OutputID, []types.Transaction, error)
	WalletDiscard(txn types.Transaction) error
	WalletSign(txn *types.Transaction, toSign []types.OutputID, cf types.CoveredFields) error

	// hostdb
	AllHosts() ([]hostdb.Host, error)
	Hosts(notSince time.Time, max int) ([]hostdb.Host, error)
	Host(hostKey consensus.PublicKey) (hostdb.Host, error)
	RecordHostInteraction(hostKey consensus.PublicKey, hi hostdb.Interaction) error

	// contracts
	AddContract(c rhpv2.Contract) error
	RenewableContracts(renewWindow uint64) ([]bus.Contract, error)
	AcquireContractLock(types.FileContractID) (types.FileContractRevision, error)
	ReleaseContractLock(types.FileContractID) error

	// contractsets
	SetHostSet(name string, hosts []consensus.PublicKey) error
	HostSetContracts(name string) ([]bus.Contract, error)
}

type Worker interface {
	RHPScan(hostKey consensus.PublicKey, hostIP string) (worker.RHPScanResponse, error)
	RHPPrepareForm(renterKey consensus.PrivateKey, hostKey consensus.PublicKey, renterFunds types.Currency, renterAddress types.UnlockHash, hostCollateral types.Currency, endHeight uint64, hostSettings rhpv2.HostSettings) (types.FileContract, types.Currency, error)
	RHPPrepareRenew(contract types.FileContractRevision, renterKey consensus.PrivateKey, hostKey consensus.PublicKey, renterFunds types.Currency, renterAddress types.UnlockHash, hostCollateral types.Currency, endHeight uint64, hostSettings rhpv2.HostSettings) (types.FileContract, types.Currency, types.Currency, error)
	RHPForm(renterKey consensus.PrivateKey, hostKey consensus.PublicKey, hostIP string, transactionSet []types.Transaction) (rhpv2.Contract, []types.Transaction, error)
	RHPRenew(renterKey consensus.PrivateKey, hostKey consensus.PublicKey, hostIP string, contractID types.FileContractID, transactionSet []types.Transaction, finalPayment types.Currency) (rhpv2.Contract, []types.Transaction, error)
}

type Autopilot struct {
	store     Store
	bus       Bus
	worker    Worker
	masterKey [32]byte

	stopChan chan struct{}
}

// Config returns the autopilot's current configuration.
func (ap *Autopilot) Config() Config {
	return ap.store.Config()
}

// SetConfig updates the autopilot's configuration.
func (ap *Autopilot) SetConfig(c Config) error {
	return ap.store.SetConfig(c)
}

// Actions returns the autopilot actions that have occurred since the given time.
func (ap *Autopilot) Actions(since time.Time, max int) []Action {
	panic("unimplemented")
}

func (ap *Autopilot) Run() error {
	go ap.contractLoop()
	go ap.hostScanLoop()
	<-ap.stopChan
	return nil // TODO
}

func (ap *Autopilot) Stop() {
	close(ap.stopChan)
}

// New initializes an Autopilot.
func New(store Store, bus Bus, worker Worker) (*Autopilot, error) {
	return &Autopilot{
		store:  store,
		bus:    bus,
		worker: worker,

		stopChan: make(chan struct{}),
	}, nil
}

func (ap *Autopilot) configHandlerGET(jc jape.Context) {
	jc.Encode(ap.Config())
}

func (ap *Autopilot) configHandlerPUT(jc jape.Context) {
	var c Config
	if jc.Decode(&c) == nil {
		ap.SetConfig(c)
	}
}

func (ap *Autopilot) actionsHandler(jc jape.Context) {
	var since time.Time
	max := -1
	if jc.DecodeForm("since", (*paramTime)(&since)) != nil || jc.DecodeForm("max", &max) != nil {
		return
	}
	jc.Encode(ap.Actions(since, max))
}

// NewServer returns an HTTP handler that serves the renterd autopilot API.
func NewServer(ap *Autopilot) http.Handler {
	return jape.Mux(map[string]jape.Handler{
		"GET    /config":  ap.configHandlerGET,
		"PUT    /config":  ap.configHandlerPUT,
		"GET    /actions": ap.actionsHandler,
	})
}
