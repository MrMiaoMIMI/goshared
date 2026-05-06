package mqhelper

import "testing"

func TestNewProducerMessageWithHeaders(t *testing.T) {
	msg := NewProducerMessageWithHeaders("topic-a", []byte("key-a"), []byte("value-a"), map[string][]byte{
		"trace-id": []byte("trace-a"),
	})

	if msg.Topic != "topic-a" {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if string(msg.Key) != "key-a" || string(msg.Value) != "value-a" {
		t.Fatalf("unexpected key/value")
	}
	if len(msg.Headers) != 1 || string(msg.Headers[0].Key) != "trace-id" {
		t.Fatalf("unexpected headers: %#v", msg.Headers)
	}
}

func TestNewJSONProducerMessage(t *testing.T) {
	msg, err := NewJSONProducerMessage("topic-a", []byte("key-a"), map[string]string{"name": "alice"})
	if err != nil {
		t.Fatalf("new json producer message failed: %v", err)
	}
	if string(msg.Value) != `{"name":"alice"}` {
		t.Fatalf("unexpected json value: %s", string(msg.Value))
	}
}
