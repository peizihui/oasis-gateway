package core

import (
	"encoding/json"
	"fmt"

	"github.com/oasislabs/oasis-gateway/errors"
	mqueue "github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/rpc"
)

type Event interface {
	EventID() uint64
	EventType() EventType
}

type Events struct {
	Offset uint64
	Events []Event
}

type EventType string

const (
	DeployServiceEventType  EventType = "deployServiceEventType"
	ExecuteServiceEventType EventType = "executeServiceEventType"
	ErrorEventType          EventType = "errorEventType"
	DataEventType           EventType = "dataEventType"
)

func (t EventType) String() string {
	return string(t)
}

func makeElement(ev Event, offset uint64) (mqueue.Element, error) {
	p, err := json.Marshal(ev)
	if err != nil {
		return mqueue.Element{}, err
	}

	return mqueue.Element{
		Offset: offset,
		Type:   ev.EventType().String(),
		Value:  string(p),
	}, nil
}

func deserializeElement(el mqueue.Element) (Event, errors.Err) {
	switch EventType(el.Type) {
	case DeployServiceEventType:
		var ev DeployServiceResponse
		if err := json.Unmarshal([]byte(el.Value), &ev); err != nil {
			return nil, errors.New(errors.ErrDeserializeEvent, err)
		}

		return ev, nil
	case ExecuteServiceEventType:
		var ev ExecuteServiceResponse
		if err := json.Unmarshal([]byte(el.Value), &ev); err != nil {
			return nil, errors.New(errors.ErrDeserializeEvent, err)
		}

		return ev, nil
	case ErrorEventType:
		var ev ErrorEvent
		if err := json.Unmarshal([]byte(el.Value), &ev); err != nil {
			return nil, errors.New(errors.ErrDeserializeEvent, err)
		}

		return ev, nil
	case DataEventType:
		var ev DataEvent
		if err := json.Unmarshal([]byte(el.Value), &ev); err != nil {
			return nil, errors.New(errors.ErrDeserializeEvent, err)
		}

		return ev, nil
	default:
		return nil, errors.New(errors.ErrUnkownEventType, nil)
	}
}

// SubID generates a subscription ID that uniquely
// identifies a subscription within the global namespace
func SubID(key string, id uint64) string {
	return fmt.Sprintf("%s:sub:%d", key, id)
}

// SubinfoID generates the ID that uniquely identifies
// the managed subscriptions of a session
func SubinfoID(key string) string {
	return fmt.Sprintf("%s:subinfo", key)
}

// ExecuteServiceRequest is is used by the user to trigger a service
// execution. A client is always subscribed to a subscription with
// topic "service" from which the client can retrieve the asynchronous
// results to this request
type ExecuteServiceRequest struct {
	// AAD is the identifier of the issuer of the transaction data
	AAD string

	// Data is a blob of data that the user wants to pass to the service
	// as argument
	Data string

	// Address where the service can be found
	Address string

	// Key is the identifier of the session
	SessionKey string
}

// DeployServiceRequest is issued by the user to trigger a service
// execution. A client is always subscribed to a subscription with
// topic "service" from which the client can retrieve the asynchronous
// results to this request
type DeployServiceRequest struct {
	// AAD is the identifier of the issuer of the transaction data
	AAD string

	// Data is a blob of data that the user wants to pass as argument for
	// the deployment of a service
	Data string

	// Key is the identifier of the session
	SessionKey string
}

// GetCodeRequest is a request to retrieve the code
// associated with a specific service
type GetCodeRequest struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// GetCodeResponse is the response in which the code
// associated with the service is provided
type GetCodeResponse struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string

	// Code associated to the service
	Code string
}

// GetPublicKeyRequest is a request to retrieve the public key
// associated with a specific service
type GetPublicKeyRequest struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// GetPublicKeyResponse is the response in which the public key
// associated with the service is provided
type GetPublicKeyResponse struct {
	// Timestamp at which the key expired
	Timestamp uint64

	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string

	// PublicKey associated to the service
	PublicKey string

	// Signature from the key manager to authenticate the public key
	Signature string
}

// ErrorEvent is the event that can be polled by the user
// as a result to a a request that failed
type ErrorEvent struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64

	// Cause is the error that caused the event to failed
	Cause rpc.Error
}

// ExecuteServiceResponse is the event that can be polled by the user
// as a result to a ServiceExecutionRequest
type ExecuteServiceResponse struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64

	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string

	// Output generated by the service at the end of its execution
	Output string
}

// DeployServiceResponse is the event that can be polled by the user
// as a result to a ServiceDeployRequest
type DeployServiceResponse struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64

	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string
}

// DataEvent is that event that can be polled by the user to poll
// for service logs for example, which they are a blob of data that the
// client knows how to manipulate
type DataEvent struct {
	// ID to identify the event itself within the sequence of events.
	ID uint64

	// Data is the blob of data related to this event
	Data string

	// Topics is the list of topics to which this event refers
	Topics []string
}

// EventID is the implementation of Event for ExecuteServiceResponse
func (e ExecuteServiceResponse) EventID() uint64 {
	return e.ID
}

// EventType is the implementation of Event for ExecuteServiceResponse
func (e ExecuteServiceResponse) EventType() EventType {
	return ExecuteServiceEventType
}

// EventID is the implementation of rpc.Event for DeployServiceResponse
func (e DeployServiceResponse) EventID() uint64 {
	return e.ID
}

// EventType is the implementation of Event for DeployServiceResponse
func (e DeployServiceResponse) EventType() EventType {
	return DeployServiceEventType
}

// EventID is the implementation of rpc.Event for ErrorEvent
func (e ErrorEvent) EventID() uint64 {
	return e.ID
}

// EventType is the implementation of Event for ErrorResponse
func (e ErrorEvent) EventType() EventType {
	return ErrorEventType
}

// EventID is the implementation of rpc.Event for DataEvent
func (e DataEvent) EventID() uint64 {
	return e.ID
}

// EventType is the implementation of Event for ErrorResponse
func (e DataEvent) EventType() EventType {
	return DataEventType
}

// PollServiceRequest is a request issued by a client to
// retrieve a window of responses generated by
// asynchronous requests
type PollServiceRequest struct {
	// Offset at which events need to be provided. Events are all ordered
	// with sequence numbers and it is up to the client to specify which
	// events it wants to receive from an offset in the sequence
	Offset uint64

	// Count for the number of items the client would prefer to receive
	// at most from a single response
	Count uint

	// DiscardPrevious allows the client to define whether the server should
	// discard all the events that have a sequence number lower than the offer
	DiscardPrevious bool

	// Key is the identifier of the request issuer
	SessionKey string
}

// SubscribeRequest is a request issued by the client to subscribe to a
// specific event type and receive events from it until the subscription is
// closed
type SubscribeRequest struct {
	// Event is the subscription event to subscribe to
	Event string

	// Address will be used to filter events only issues by or to
	// the address
	Address string

	// Key is the identifier of the session
	SessionKey string

	// Topics is the list of topics the subscription client is
	// interested in
	Topics []string
}

// PollEventRequest is a request issued by the client to
// poll events from an already created subscription
type PollEventRequest struct {
	// Offset at which events need to be provided. Events are all ordered
	// with sequence numbers and it is up to the client to specify which
	// events it wants to receive from an offset in the sequence
	Offset uint64

	// Count for the number of items the client would prefer to receive
	// at most from a single response
	Count uint

	// DiscardPrevious allows the client to define whether the server should
	// discard all the events that have a sequence number lower than the offer
	DiscardPrevious bool

	// ID is the unique identifier for a subscription based on
	// the user's key namespace
	ID uint64

	// Key is the identifier of the session
	SessionKey string
}

// UnsubscribeRequest is a request issued by the client to subscribe to a
// specific topic and receive events from it until the subscription is
// closed
type UnsubscribeRequest struct {
	// ID is the unique identifier for a subscription based on
	// the user's key namespace
	ID uint64

	// Key is the identifier of the session
	SessionKey string
}

// CreateSubscriptionRequest is the request to subscribe to a specific
// event type for a contract
type CreateSubscriptionRequest struct {
	// Event is the subscription event type
	Event string

	// Address will be used to filter events only issues by or to
	// the address
	Address string

	// SubID is the unique subscription's identifier
	SubID string

	// Topics is the list of topics the client is interested in
	Topics []string
}

// UnsubscribeRequest is a request issued by the client to destroy
// an existing subscription
type DestroySubscriptionRequest struct {
	// SubID is the unique subscription's identifier
	SubID string
}
