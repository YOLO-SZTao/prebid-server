package request_logger

import (
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleRawBidderResponse(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}

	if !m.cfg.Enabled || !m.cfg.LogRawBidderResponse {
		return result, nil
	}

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getTraceOrRequestID(miCtx.ModuleContext),
		"bidder":     payload.Bidder,
	}

	if payload.BidderResponse != nil {
		fields["bid_count"] = len(payload.BidderResponse.Bids)
		fields["currency"] = payload.BidderResponse.Currency
		fields["seat_bid_count"] = countDistinctSeats(payload.BidderResponse)
		fields["bid_ids"] = getBidIDs(payload.BidderResponse)
		fields["imp_ids"] = getBidImpIDs(payload.BidderResponse)
	}

	fields["bidder_window_elapsed_ms"] = getBidderElapsedMs(miCtx.ModuleContext, payload.Bidder)

	logStage("raw_bidder_response", fields)

	return result, nil
}

func countDistinctSeats(resp *adapters.BidderResponse) int {
	if resp == nil {
		return 0
	}
	seats := make(map[string]bool)
	for _, bid := range resp.Bids {
		if bid != nil {
			seats[string(bid.Seat)] = true
		}
	}
	return len(seats)
}
