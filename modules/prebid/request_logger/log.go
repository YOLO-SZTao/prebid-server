package request_logger

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v4/logger"
)

const componentName = "pbs-hook-request-logger"

func logStage(stage string, fields map[string]any) {
	fields["component"] = componentName
	fields["stage"] = stage

	data, err := json.Marshal(fields)
	if err != nil {
		logger.Infof(`{"component":"%s","stage":"%s","error":"marshal_failed"}`, componentName, stage)
		return
	}
	logger.Infof("%s", string(data))
}
