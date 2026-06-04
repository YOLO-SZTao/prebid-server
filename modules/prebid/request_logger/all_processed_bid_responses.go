package request_logger

import (
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleAllProcessedBidResponses(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AllProcessedBidResponsesPayload,
) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	result := hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}

	if !m.cfg.Enabled || !m.cfg.LogAllProcessedBidResponses {
		return result, nil
	}

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getTraceOrRequestID(miCtx.ModuleContext),
	}

	bidderNames := collectBidderNamesFromSeatBids(payload.Responses)
	fields["bidders_with_bids"] = bidderNames
	fields["seat_bid_count"] = len(payload.Responses)
	fields["total_bids"] = countTotalBidsFromSeatBids(payload.Responses)

	logStage("all_processed_bid_responses", fields)

	return result, nil
}
