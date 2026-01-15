package errors

import (
	pkgerrors "github.com/Conte777/NewsFlow/services/account-service/pkg/errors"
)

var (
	ErrNewsCollectionFailed = pkgerrors.NewInternalError("news collection failed")
	ErrKafkaSendFailed      = pkgerrors.NewInternalError("failed to send to Kafka")
)
