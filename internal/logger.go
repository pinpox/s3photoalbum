package s3photoalbum

import (
	"go.uber.org/zap"
)

func NewLogger(devMode bool) *zap.SugaredLogger {

	var logger *zap.Logger
	var err error

	if devMode {
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(err)
		}

	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}

	defer logger.Sync()
	return logger.Sugar()
}
