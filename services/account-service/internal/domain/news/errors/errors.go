package errors

import (
	pkgerrors "github.com/YarosTrubechkoi/telegram-news-feed/account-service/pkg/errors"
)

var (
	ErrNewsCollectionFailed = pkgerrors.NewInternalError("news collection failed")
	ErrKafkaSendFailed      = pkgerrors.NewInternalError("failed to send to Kafka")
)
