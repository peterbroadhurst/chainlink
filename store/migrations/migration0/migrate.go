package migration0

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/smartcontractkit/chainlink/store/models"
	"github.com/smartcontractkit/chainlink/store/orm"
	"github.com/tidwall/gjson"
	null "gopkg.in/guregu/null.v3"
)

type Migration struct{}

func (m Migration) Timestamp() string {
	return "0"
}

func (m Migration) Migrate(orm *orm.ORM) error {
	orm.InitializeModel(&JobSpec{})
	orm.InitializeModel(&JobRun{})
	orm.InitializeModel(&Initiator{})
	orm.InitializeModel(&Tx{})
	orm.InitializeModel(&TxAttempt{})
	orm.InitializeModel(&BridgeType{})
	orm.InitializeModel(&IndexableBlockNumber{})
	orm.InitializeModel(&User{})
	orm.InitializeModel(&Session{})
	orm.InitializeModel(&ServiceAgreement{})
	return nil
}

type Time struct {
	time.Time
}

// UnmarshalJSON parses the raw time stored in JSON-encoded
// data and stores it to the Time field.
func (t *Time) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	newTime, err := dateparse.ParseAny(n.String())
	t.Time = newTime.UTC()
	return err
}

type JobSpec struct {
	ID        string `json:"id" storm:"id,unique"`
	CreatedAt Time   `json:"createdAt" storm:"index"`
	JobSpecRequest
}

// JobSpecRequest represents a schema for the incoming job spec request as used by the API.
type JobSpecRequest struct {
	Initiators []Initiator `json:"initiators"`
	Tasks      []TaskSpec  `json:"tasks" storm:"inline"`
	StartAt    null.Time   `json:"startAt" storm:"index"`
	EndAt      null.Time   `json:"endAt" storm:"index"`
}

type RunStatus string

type RunResult struct {
	JobRunID     string      `json:"jobRunId"`
	Data         models.JSON `json:"data"`
	Status       RunStatus   `json:"status"`
	ErrorMessage null.String `json:"error"`
	Amount       *big.Int    `json:"amount,omitempty"`
}

type TaskType string

type TaskSpec struct {
	Type          TaskType    `json:"type" storm:"index"`
	Confirmations uint64      `json:"confirmations"`
	Params        models.JSON `json:"-"`
}

// UnmarshalJSON parses the given input and updates the TaskSpec.
func (t *TaskSpec) UnmarshalJSON(input []byte) error {
	type Alias TaskSpec
	var aux Alias
	if err := json.Unmarshal(input, &aux); err != nil {
		return err
	}

	t.Confirmations = aux.Confirmations
	t.Type = aux.Type
	var params json.RawMessage
	if err := json.Unmarshal(input, &params); err != nil {
		return err
	}

	t.Params = models.JSON{gjson.ParseBytes(params)}
	return nil
}

// MarshalJSON returns the JSON-encoded TaskSpec Params.
func (t TaskSpec) MarshalJSON() ([]byte, error) {
	type Alias TaskSpec
	var aux Alias
	aux = Alias(t)
	b, err := json.Marshal(aux)
	if err != nil {
		return b, err
	}

	js := gjson.ParseBytes(b)
	merged, err := t.Params.Merge(models.JSON{js})
	if err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

type TaskRun struct {
	ID     string    `json:"id" storm:"id,unique"`
	Result RunResult `json:"result"`
	Status RunStatus `json:"status"`
	Task   TaskSpec  `json:"task"`
}

type JobRun struct {
	ID             string       `json:"id" storm:"id,unique"`
	JobID          string       `json:"jobId" storm:"index"`
	Result         RunResult    `json:"result" storm:"inline"`
	Status         RunStatus    `json:"status" storm:"index"`
	TaskRuns       []TaskRun    `json:"taskRuns" storm:"inline"`
	CreatedAt      time.Time    `json:"createdAt" storm:"index"`
	CompletedAt    null.Time    `json:"completedAt"`
	Initiator      Initiator    `json:"initiator"`
	CreationHeight *hexutil.Big `json:"creationHeight"`
	Overrides      RunResult    `json:"overrides"`
}

type Cron string

type Initiator struct {
	ID       int            `json:"id" storm:"id,increment"`
	JobID    string         `json:"jobId" storm:"index"`
	Type     string         `json:"type" storm:"index"`
	Schedule Cron           `json:"schedule,omitempty"`
	Time     Time           `json:"time,omitempty"`
	Ran      bool           `json:"ran,omitempty"`
	Address  common.Address `json:"address,omitempty" storm:"index"`
}

func (i *Initiator) UnmarshalJSON(input []byte) error {
	type Alias Initiator
	var aux Alias
	if err := json.Unmarshal(input, &aux); err != nil {
		return err
	}

	*i = Initiator(aux)
	i.Type = strings.ToLower(aux.Type)
	return nil
}

type Tx struct {
	ID       uint64         `storm:"id,increment,index"`
	From     common.Address `storm:"index"`
	To       common.Address
	Data     []byte
	Nonce    uint64 `storm:"index"`
	Value    *big.Int
	GasLimit uint64
	TxAttempt
}

type TxAttempt struct {
	Hash      common.Hash `storm:"id,unique"`
	TxID      uint64      `storm:"index"`
	GasPrice  *big.Int
	Confirmed bool
	Hex       string
	SentAt    uint64
}

type BridgeType struct {
	Name                 TaskType `json:"name" storm:"id,unique"`
	URL                  WebURL   `json:"url"`
	DefaultConfirmations uint64   `json:"defaultConfirmations"`
	IncomingToken        string   `json:"incomingToken"`
	OutgoingToken        string   `json:"outgoingToken"`
}

type IndexableBlockNumber struct {
	Number hexutil.Big `json:"number" storm:"id,unique"`
	Digits int         `json:"digits" storm:"index"`
	Hash   common.Hash `json:"hash"`
}

type User struct {
	Email          string `json:"email" storm:"id,unique"`
	HashedPassword string `json:"hashedPassword"`
	CreatedAt      Time   `json:"createdAt" storm:"index"`
}

type Session struct {
	ID       string `json:"id" storm:"id,unique"`
	LastUsed Time   `json:"lastUsed" storm:"index"`
}

type EIP55Address string

type Encumbrance struct {
	Payment    *Link          `json:"payment"`
	Expiration uint64         `json:"expiration"`
	Oracles    []EIP55Address `json:"oracles"`
}

const SignatureLength = 65

type Signature [SignatureLength]byte

type ServiceAgreement struct {
	CreatedAt   Time        `json:"createdAt" storm:"index"`
	Encumbrance Encumbrance `json:"encumbrance" storm:"inline"`
	ID          string      `json:"id" storm:"id,unique"`
	JobSpecID   string      `json:"jobSpecID"`
	RequestBody string      `json:"requestBody"`
	Signature   Signature   `json:"signature"`
	JobSpec     JobSpec
}

// WebURL contains the URL of the endpoint.
type WebURL struct {
	*url.URL
}

// UnmarshalJSON parses the raw URL stored in JSON-encoded
// data to a URL structure and sets it to the URL field.
func (w *WebURL) UnmarshalJSON(j []byte) error {
	var v string
	err := json.Unmarshal(j, &v)
	if err != nil {
		return err
	}
	u, err := url.ParseRequestURI(v)
	if err != nil {
		return err
	}
	w.URL = u
	return nil
}

type Link big.Int

// SetString delegates to *big.Int.SetString
func (l *Link) SetString(s string, base int) (*Link, bool) {
	w, ok := (*big.Int)(l).SetString(s, base)
	return (*Link)(w), ok
}

// MarshalText implements the encoding.TextMarshaler interface.
func (l *Link) MarshalText() ([]byte, error) {
	return (*big.Int)(l).MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (l *Link) UnmarshalText(text []byte) error {
	if _, ok := l.SetString(string(text), 10); !ok {
		return fmt.Errorf("assets: cannot unmarshal %q into a *assets.Link", text)
	}
	return nil
}
