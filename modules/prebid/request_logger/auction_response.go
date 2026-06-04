package request_logger

import (
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *RequestLoggerModule) handleAuctionResponse(
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	result := hookstage.HookResult[hookstage.AuctionResponsePayload]{}

	if !m.cfg.Enabled || !m.cfg.LogAuctionResponse {
		return result, nil
	}

	fields := map[string]any{
		"trace_id":   getTraceID(miCtx.ModuleContext),
		"request_id": getTraceOrRequestID(miCtx.ModuleContext),
	}

	if payload.BidResponse != nil {
		fields["response_id"] = payload.BidResponse.ID
		fields["seat_bid_count"] = countSeatBids(payload.BidResponse)
		fields["total_bids"] = countBids(payload.BidResponse)
		fields["currency"] = payload.BidResponse.Cur
		if payload.BidResponse.NBR != nil {
			fields["nbr"] = *payload.BidResponse.NBR
		}

		errorBidders, errorDetails, warningBidders, warningDetails, httpcallBidders := summarizeExt(payload.BidResponse)
		fields["ext_error_bidders"] = errorBidders
		fields["ext_error_details"] = errorDetails
		fields["ext_warning_bidders"] = warningBidders
		fields["ext_warning_details"] = warningDetails
		fields["httpcall_bidders"] = httpcallBidders
			fields["httpcall_details"] = collectHttpCallDetails(payload.BidResponse, m.cfg.MaxPayloadLogSize)
	}

	logStage("auction_response", fields, m.cfg.LogPrettyPrint)

	return result, nil
}
