package request_logger

import (
	"sync"
	"time"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleEntrypoint(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	result := hookstage.HookResult[hookstage.EntrypointPayload]{}

	if !m.cfg.Enabled {
		return result, nil
	}

	traceID := generateTraceID()

	mc := hookstage.NewModuleContext()
	mc.Set(ctxKeyTraceID, traceID)
	mc.Set(ctxKeyStartTime, time.Now())
	mc.Set(ctxKeyBidderTimers, &sync.Map{})
	result.ModuleContext = mc

	if !m.cfg.LogEntrypoint {
		return result, nil
	}

	fields := map[string]any{
		"trace_id": traceID,
	}

	if payload.Request != nil {
		fields["method"] = payload.Request.Method
		fields["path"] = payload.Request.URL.Path
		fields["query"] = payload.Request.URL.RawQuery
		fields["content_type"] = payload.Request.Header.Get("Content-Type")
		fields["headers"] = headersToMap(payload.Request.Header)
	}

	fields["body_size"] = len(payload.Body)
	if len(payload.Body) > 0 {
		fields["body_preview"] = previewBytes(payload.Body, m.cfg.MaxPayloadLogSize)
	}

	logStage("entrypoint", fields, m.cfg.LogPrettyPrint)

	return result, nil
}
