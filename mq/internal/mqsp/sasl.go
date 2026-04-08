package mqsp

import (
	"crypto/sha256"
	"crypto/sha512"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
	"github.com/xdg-go/scram"
)

// applySASL configures SASL authentication on a sarama config if credentials are provided.
func applySASL(saramaConfig *sarama.Config, credentials mqspi.Credentials) {
	if credentials == nil || credentials.Mechanism() == "" {
		return
	}

	saramaConfig.Net.SASL.Enable = true
	saramaConfig.Net.SASL.User = credentials.Username()
	saramaConfig.Net.SASL.Password = credentials.Password()

	switch credentials.Mechanism() {
	case "PLAIN":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	case "SCRAM-SHA-256":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &scramClient{HashGeneratorFcn: scram.HashGeneratorFcn(sha256.New)}
		}
	case "SCRAM-SHA-512":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &scramClient{HashGeneratorFcn: scram.HashGeneratorFcn(sha512.New)}
		}
	}
}

// scramClient implements sarama.SCRAMClient using xdg-go/scram.
type scramClient struct {
	HashGeneratorFcn scram.HashGeneratorFcn
	conversation     *scram.ClientConversation
}

func (c *scramClient) Begin(userName, password, authzID string) error {
	client, err := c.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	c.conversation = client.NewConversation()
	return nil
}

func (c *scramClient) Step(challenge string) (string, error) {
	return c.conversation.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.conversation.Done()
}
