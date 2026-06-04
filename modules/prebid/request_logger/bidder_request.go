package request_logger

import (
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleBidderRequest(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	result := hookstage.HookResult[hookstage.BidderRequestPayload]{}

	if !m.cfg.Enabled || !m.cfg.LogBidderRequest {
		return result, nil
	}

	recordBidderStartTime(miCtx.ModuleContext, payload.Bidder)

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getTraceOrRequestID(miCtx.ModuleContext),
		"bidder":     payload.Bidder,
	}

	if payload.Request != nil {
		fields["imp_ids"] = getImpIDs(payload.Request)
		fields["imp_count"] = payload.Request.LenImp()

		if payload.Request.TMax > 0 {
			fields["tmax_ms"] = payload.Request.TMax
		}

		fields["request_preview"] = previewJSON(payload.Request.BidRequest, m.cfg.MaxPayloadLogSize)
	}

	logStage("bidder_request", fields, m.cfg.LogPrettyPrint)

	return result, nil
}
