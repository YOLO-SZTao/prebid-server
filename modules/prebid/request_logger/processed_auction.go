package request_logger

import (
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleProcessedAuction(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}

	if !m.cfg.Enabled || !m.cfg.LogProcessedAuction {
		return result, nil
	}

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getRequestID(miCtx.ModuleContext),
		"account_id": miCtx.AccountID,
	}

	if payload.Request != nil {
		reqID := payload.Request.ID
		if reqID != "" {
			if getRequestID(miCtx.ModuleContext) == "" {
				if result.ModuleContext == nil {
					result.ModuleContext = hookstage.NewModuleContext()
				}
				result.ModuleContext.Set(ctxKeyRequestID, reqID)
			}
			fields["request_id"] = reqID
		}

		fields["imp_count"] = payload.Request.LenImp()
		fields["bidders"] = getBidderNames(payload.Request)
		fields["source"] = getSource(payload.Request)

		if payload.Request.TMax > 0 {
			fields["tmax_ms"] = payload.Request.TMax
		}

		fields["request_preview"] = previewJSON(payload.Request.BidRequest, m.cfg.MaxPayloadLogSize)
	}

	logStage("processed_auction", fields)

	return result, nil
}
