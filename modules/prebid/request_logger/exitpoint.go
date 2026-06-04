package request_logger

import (
	"fmt"
	"time"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleExitpoint(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ExitpointPayload,
) (hookstage.HookResult[hookstage.ExitpointPayload], error) {
	result := hookstage.HookResult[hookstage.ExitpointPayload]{}

	if !m.cfg.Enabled || !m.cfg.LogExitpoint {
		return result, nil
	}

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getTraceOrRequestID(miCtx.ModuleContext),
	}

	fields["response_type"] = fmt.Sprintf("%T", payload.Response)

	if startTime, ok := getStartTime(miCtx.ModuleContext); ok {
		fields["request_elapsed_ms"] = time.Since(startTime).Milliseconds()
	}

	logStage("exitpoint", fields)

	return result, nil
}
