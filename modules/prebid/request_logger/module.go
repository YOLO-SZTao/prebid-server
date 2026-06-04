package request_logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Builder returns a new RequestLoggerModule instance.
// It is registered in modules/builder.go via go generate.
func Builder(cfg json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	mCfg := newConfig()
	if len(cfg) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(cfg, &raw); err != nil {
			return nil, err
		}
		if enabled, ok := raw["enabled"].(bool); ok {
			mCfg.Enabled = enabled
		}
		if v, ok := raw["log_entrypoint"].(bool); ok {
			mCfg.LogEntrypoint = v
		}
		if v, ok := raw["log_processed_auction"].(bool); ok {
			mCfg.LogProcessedAuction = v
		}
		if v, ok := raw["log_bidder_request"].(bool); ok {
			mCfg.LogBidderRequest = v
		}
		if v, ok := raw["log_raw_bidder_response"].(bool); ok {
			mCfg.LogRawBidderResponse = v
		}
		if v, ok := raw["log_all_processed_bid_responses"].(bool); ok {
			mCfg.LogAllProcessedBidResponses = v
		}
		if v, ok := raw["log_auction_response"].(bool); ok {
			mCfg.LogAuctionResponse = v
		}
		if v, ok := raw["log_exitpoint"].(bool); ok {
			mCfg.LogExitpoint = v
		}
		if v, ok := raw["max_payload_log_size"].(float64); ok {
			mCfg.MaxPayloadLogSize = int(v)
		}
	}
	validateConfig(&mCfg)
	return &RequestLoggerModule{cfg: mCfg}, nil
}

// RequestLoggerModule implements 7 hook stage interfaces for auction request logging.
type RequestLoggerModule struct {
	cfg RequestLoggerConfig
}

func generateTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "hook-fallback"
	}
	return "hook-" + hex.EncodeToString(b)
}

// HandleEntrypointHook implements hookstage.Entrypoint
func (m *RequestLoggerModule) HandleEntrypointHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return m.handleEntrypoint(miCtx, payload)
}

// HandleProcessedAuctionHook implements hookstage.ProcessedAuctionRequest
func (m *RequestLoggerModule) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	return m.handleProcessedAuction(miCtx, payload)
}

// HandleBidderRequestHook implements hookstage.BidderRequest
func (m *RequestLoggerModule) HandleBidderRequestHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	return m.handleBidderRequest(miCtx, payload)
}

// HandleRawBidderResponseHook implements hookstage.RawBidderResponse
func (m *RequestLoggerModule) HandleRawBidderResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	return m.handleRawBidderResponse(miCtx, payload)
}

// HandleAllProcessedBidResponsesHook implements hookstage.AllProcessedBidResponses
func (m *RequestLoggerModule) HandleAllProcessedBidResponsesHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AllProcessedBidResponsesPayload,
) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return m.handleAllProcessedBidResponses(miCtx, payload)
}

// HandleAuctionResponseHook implements hookstage.AuctionResponse
func (m *RequestLoggerModule) HandleAuctionResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return m.handleAuctionResponse(miCtx, payload)
}

// HandleExitpointHook implements hookstage.Exitpoint
func (m *RequestLoggerModule) HandleExitpointHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ExitpointPayload,
) (hookstage.HookResult[hookstage.ExitpointPayload], error) {
	return m.handleExitpoint(miCtx, payload)
}
