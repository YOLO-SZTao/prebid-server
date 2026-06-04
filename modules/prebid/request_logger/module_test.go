package request_logger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	t.Run("empty config uses defaults", func(t *testing.T) {
		mod, err := Builder(json.RawMessage{}, moduledeps.ModuleDeps{})
		assert.NoError(t, err)
		m := mod.(*RequestLoggerModule)
		assert.True(t, m.cfg.Enabled)
		assert.True(t, m.cfg.LogEntrypoint)
		assert.Equal(t, 2048, m.cfg.MaxPayloadLogSize)
	})

	t.Run("disabled", func(t *testing.T) {
		cfg := json.RawMessage(`{"enabled":false}`)
		mod, err := Builder(cfg, moduledeps.ModuleDeps{})
		assert.NoError(t, err)
		m := mod.(*RequestLoggerModule)
		assert.False(t, m.cfg.Enabled)
	})

	t.Run("custom config", func(t *testing.T) {
		cfg := json.RawMessage(`{"enabled":true,"log_entrypoint":false,"max_payload_log_size":1024}`)
		mod, err := Builder(cfg, moduledeps.ModuleDeps{})
		assert.NoError(t, err)
		m := mod.(*RequestLoggerModule)
		assert.True(t, m.cfg.Enabled)
		assert.False(t, m.cfg.LogEntrypoint)
		assert.Equal(t, 1024, m.cfg.MaxPayloadLogSize)
	})

	t.Run("invalid max_payload_log_size falls back to default", func(t *testing.T) {
		cfg := json.RawMessage(`{"max_payload_log_size":0}`)
		mod, err := Builder(cfg, moduledeps.ModuleDeps{})
		assert.NoError(t, err)
		m := mod.(*RequestLoggerModule)
		assert.Equal(t, 2048, m.cfg.MaxPayloadLogSize)
	})
}

func TestHandleEntrypoint(t *testing.T) {
	t.Run("disabled module returns nil ModuleContext", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: false}}
		result, err := m.HandleEntrypointHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.EntrypointPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
		assert.Nil(t, result.ModuleContext)
	})

	t.Run("enabled with log_entrypoint=false still initializes ModuleContext", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{
			Enabled:       true,
			LogEntrypoint: false,
		}}
		result, err := m.HandleEntrypointHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.EntrypointPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
		assert.NotNil(t, result.ModuleContext)

		traceID := getTraceID(result.ModuleContext)
		assert.NotEmpty(t, traceID)
		assert.Contains(t, traceID, "hook-")

		_, ok := getStartTime(result.ModuleContext)
		assert.True(t, ok)

		timers := getBidderTimers(result.ModuleContext)
		assert.NotNil(t, timers)
	})

	t.Run("enabled initializes ModuleContext", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/openrtb2/auction", RawQuery: "debug=1"},
			Header: http.Header{"Content-Type": {"application/json"}},
		}
		payload := hookstage.EntrypointPayload{Request: req, Body: []byte(`{"id":"test"}`)}
		result, err := m.HandleEntrypointHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			payload)
		assert.NoError(t, err)
		assert.False(t, result.Reject)

		assert.NotNil(t, result.ModuleContext)
		traceID := getTraceID(result.ModuleContext)
		assert.NotEmpty(t, traceID)
		assert.Contains(t, traceID, "hook-")

		_, ok := getStartTime(result.ModuleContext)
		assert.True(t, ok)

		timers := getBidderTimers(result.ModuleContext)
		assert.NotNil(t, timers)
	})

	t.Run("empty body", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		result, err := m.HandleEntrypointHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.EntrypointPayload{})
		assert.NoError(t, err)
		assert.NotNil(t, result.ModuleContext)
	})
}

func TestEntrypointContextPropagationToLaterStages(t *testing.T) {
	m := &RequestLoggerModule{cfg: RequestLoggerConfig{
		Enabled:               true,
		LogEntrypoint:         false,
		LogBidderRequest:      true,
		LogRawBidderResponse:  true,
		LogExitpoint:          true,
	}}

	entryResult, err := m.HandleEntrypointHook(context.Background(),
		hookstage.ModuleInvocationContext{},
		hookstage.EntrypointPayload{})
	assert.NoError(t, err)
	assert.NotNil(t, entryResult.ModuleContext)
	entryMC := entryResult.ModuleContext

	t.Run("bidder_request can read trace_id and record timer", func(t *testing.T) {
		_, err := m.HandleBidderRequestHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: entryMC},
			hookstage.BidderRequestPayload{
				Bidder:  "pubmatic",
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			})
		assert.NoError(t, err)

		timers := getBidderTimers(entryMC)
		assert.NotNil(t, timers)
		_, ok := timers.Load("pubmatic")
		assert.True(t, ok, "bidder timer should be recorded")
	})

	t.Run("raw_bidder_response gets elapsed ms", func(t *testing.T) {
		time.Sleep(10 * time.Millisecond)

		_, err := m.HandleRawBidderResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: entryMC},
			hookstage.RawBidderResponsePayload{Bidder: "pubmatic", BidderResponse: nil})
		assert.NoError(t, err)

		elapsed := getBidderElapsedMs(entryMC, "pubmatic")
		assert.Greater(t, elapsed, int64(0), "elapsed should be > 0")
	})

	t.Run("exitpoint gets start_time", func(t *testing.T) {
		_, ok := getStartTime(entryMC)
		assert.True(t, ok)
	})
}

func TestHandleProcessedAuction(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogProcessedAuction: false}}
		result, err := m.HandleProcessedAuctionHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.ProcessedAuctionRequestPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("writes request_id when present", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		req := &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{ID: "req-123"},
		}
		result, err := m.HandleProcessedAuctionHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.ProcessedAuctionRequestPayload{Request: req})
		assert.NoError(t, err)
		v, ok := result.ModuleContext.Get(ctxKeyRequestID)
		assert.True(t, ok)
		assert.Equal(t, "req-123", v)
	})

	t.Run("does not overwrite existing request_id", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		req := &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{ID: "new-id"},
		}
		mc := hookstage.NewModuleContext()
		mc.Set(ctxKeyRequestID, "existing-id")
		result, err := m.HandleProcessedAuctionHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: mc},
			hookstage.ProcessedAuctionRequestPayload{Request: req})
		assert.NoError(t, err)
		assert.Nil(t, result.ModuleContext)
	})
}

func TestHandleBidderRequest(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogBidderRequest: false}}
		result, err := m.HandleBidderRequestHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.BidderRequestPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("records bidder timer", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		timers := &sync.Map{}
		mc := hookstage.NewModuleContext()
		mc.Set(ctxKeyBidderTimers, timers)

		_, err := m.HandleBidderRequestHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: mc},
			hookstage.BidderRequestPayload{
				Bidder:  "pubmatic",
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			})
		assert.NoError(t, err)

		_, ok := timers.Load("pubmatic")
		assert.True(t, ok)
	})
}

func TestHandleBidderRequestConcurrency(t *testing.T) {
	m := &RequestLoggerModule{cfg: newConfig()}
	timers := &sync.Map{}
	mc := hookstage.NewModuleContext()
	mc.Set(ctxKeyTraceID, "trace-123")
	mc.Set(ctxKeyBidderTimers, timers)

	bidders := []string{"pubmatic", "rubicon", "appnexus", "bigoad", "axonix"}

	var wg sync.WaitGroup
	for _, bidder := range bidders {
		wg.Add(1)
		go func(b string) {
			defer wg.Done()
			_, err := m.HandleBidderRequestHook(context.Background(),
				hookstage.ModuleInvocationContext{ModuleContext: mc},
				hookstage.BidderRequestPayload{
					Bidder:  b,
					Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
				})
			assert.NoError(t, err)
		}(bidder)
	}
	wg.Wait()

	for _, bidder := range bidders {
		_, ok := timers.Load(bidder)
		assert.True(t, ok, "timer should exist for %s", bidder)
	}
}

func TestHandleRawBidderResponse(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogRawBidderResponse: false}}
		result, err := m.HandleRawBidderResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.RawBidderResponsePayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("nil bidder response", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		result, err := m.HandleRawBidderResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.RawBidderResponsePayload{Bidder: "pubmatic", BidderResponse: nil})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("with bids", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		resp := &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				{
					Bid:  &openrtb2.Bid{ID: "bid-1", ImpID: "imp-1"},
					Seat: "pubmatic",
				},
			},
		}
		result, err := m.HandleRawBidderResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.RawBidderResponsePayload{Bidder: "pubmatic", BidderResponse: resp})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})
}

func TestHandleAllProcessedBidResponses(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogAllProcessedBidResponses: false}}
		result, err := m.HandleAllProcessedBidResponsesHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.AllProcessedBidResponsesPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("with responses", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		responses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
			"pubmatic": {Bids: []*entities.PbsOrtbBid{{}}},
		}
		result, err := m.HandleAllProcessedBidResponsesHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.AllProcessedBidResponsesPayload{Responses: responses})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})
}

func TestHandleAuctionResponse(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogAuctionResponse: false}}
		result, err := m.HandleAuctionResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.AuctionResponsePayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("nil response", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		result, err := m.HandleAuctionResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.AuctionResponsePayload{BidResponse: nil})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("normal response", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		nbr := openrtb3.NoBidUnknownError
		resp := &openrtb2.BidResponse{
			ID:  "resp-1",
			Cur: "USD",
			NBR: &nbr,
			SeatBid: []openrtb2.SeatBid{
				{Bid: []openrtb2.Bid{{ID: "bid-1"}}},
			},
		}
		result, err := m.HandleAuctionResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.AuctionResponsePayload{BidResponse: resp})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("malformed ext does not panic", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		resp := &openrtb2.BidResponse{Ext: json.RawMessage(`invalid`)}
		result, err := m.HandleAuctionResponseHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.AuctionResponsePayload{BidResponse: resp})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})
}

func TestHandleExitpoint(t *testing.T) {
	t.Run("disabled stage", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: RequestLoggerConfig{Enabled: true, LogExitpoint: false}}
		result, err := m.HandleExitpointHook(context.Background(),
			hookstage.ModuleInvocationContext{},
			hookstage.ExitpointPayload{})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("nil response", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		result, err := m.HandleExitpointHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.ExitpointPayload{Response: nil})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})

	t.Run("bid response type", func(t *testing.T) {
		m := &RequestLoggerModule{cfg: newConfig()}
		resp := &openrtb2.BidResponse{ID: "resp-1"}
		result, err := m.HandleExitpointHook(context.Background(),
			hookstage.ModuleInvocationContext{ModuleContext: hookstage.NewModuleContext()},
			hookstage.ExitpointPayload{Response: resp})
		assert.NoError(t, err)
		assert.False(t, result.Reject)
	})
}

func TestCountDistinctSeats(t *testing.T) {
	assert.Equal(t, 0, countDistinctSeats(nil))
	assert.Equal(t, 2, countDistinctSeats(&adapters.BidderResponse{
		Bids: []*adapters.TypedBid{
			{Seat: "pubmatic"},
			{Seat: "rubicon"},
			{Seat: "pubmatic"},
		},
	}))
}

func TestGenerateTraceID(t *testing.T) {
	id := generateTraceID()
	assert.Contains(t, id, "hook-")
	assert.Greater(t, len(id), 5)
}

func TestContextHelpers(t *testing.T) {
	t.Run("nil ModuleContext", func(t *testing.T) {
		var nilMC *hookstage.ModuleContext
		assert.Empty(t, getTraceID(nilMC))
		assert.Empty(t, getRequestID(nilMC))
		assert.Empty(t, getTraceOrRequestID(nilMC))
		_, ok := getStartTime(nilMC)
		assert.False(t, ok)
		assert.Nil(t, getBidderTimers(nilMC))
	})

	t.Run("trace or request id prefers request", func(t *testing.T) {
		mc := hookstage.NewModuleContext()
		mc.Set(ctxKeyTraceID, "trace-1")
		mc.Set(ctxKeyRequestID, "req-1")
		assert.Equal(t, "req-1", getTraceOrRequestID(mc))
	})
}
