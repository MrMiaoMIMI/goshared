package mqspi

// Error defines MQ error
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
	ErrConsumerClosed     = mqErr("consumer_closed")
	ErrConsumerUpdating   = mqErr("consumer_updating")
	ErrConsumeContextDone = mqErr("consume_context_done")
	ErrProducerClosed     = mqErr("producer_closed")
)
