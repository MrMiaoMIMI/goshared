package mqspi

// Error is the typed error used for common MQ states.
type Error struct {
	Message string
}

func (e Error) Error() string {
	return "mq:" + e.Message
}

func mqErr(msg string) Error {
	return Error{Message: msg}
}

var (
	// ErrConsumerClosed means the consumer has already been closed.
	ErrConsumerClosed = mqErr("consumer_closed")
	// ErrConsumeContextDone means Consume returned because the caller context ended.
	ErrConsumeContextDone = mqErr("consume_context_done")
	// ErrProducerClosed means the producer has already been closed.
	ErrProducerClosed = mqErr("producer_closed")
	// ErrInvalidConfig means a config or message is invalid before reaching the broker.
	ErrInvalidConfig = mqErr("invalid_config")
)
