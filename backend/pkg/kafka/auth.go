package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// Auth carries optional Kafka authentication settings. Empty values mean no
// authentication, which is the local Redpanda default.
type Auth struct {
	Username  string
	Password  string
	Mechanism string // "plain", "scram-sha-256", "scram-sha-512"
	EnableTLS bool
}

// applyAuth configures SASL/TLS on a sarama config when credentials are present.
func applyAuth(cfg *sarama.Config, a Auth) {
	if a.Username == "" {
		return
	}
	if a.Mechanism == "" {
		a.Mechanism = "plain"
	}
	if !a.EnableTLS {
		a.EnableTLS = true
	}
	cfg.Net.SASL.Enable = true
	cfg.Net.SASL.User = a.Username
	cfg.Net.SASL.Password = a.Password
	switch a.Mechanism {
	case "scram-sha-256":
		cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		cfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &scramClient{hash: scram.SHA256} }
	case "scram-sha-512":
		cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		cfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &scramClient{hash: scram.SHA512} }
	default:
		cfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if a.EnableTLS {
		cfg.Net.TLS.Enable = true
		cfg.Net.TLS.Config = &tls.Config{MinVersion: tls.VersionTLS12}
	}
}

type scramClient struct {
	hash scram.HashGeneratorFcn
	*scram.Client
	*scram.ClientConversation
}

func (c *scramClient) Begin(userName, password, authzID string) (err error) {
	c.Client, err = c.hash.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	c.ClientConversation = c.NewConversation()
	return nil
}

func (c *scramClient) Step(challenge string) (response string, err error) {
	return c.ClientConversation.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.ClientConversation.Done()
}

var (
	_ = sha256.New
	_ = sha512.New
)
