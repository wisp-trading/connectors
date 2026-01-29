package requests

import (
	"github.com/wisp-trading/connectors/pkg/connectors/paradex/adaptor"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

type Service struct {
	client *adaptor.Client
	logger logging.ApplicationLogger
}

func NewService(client *adaptor.Client, logger logging.ApplicationLogger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}
