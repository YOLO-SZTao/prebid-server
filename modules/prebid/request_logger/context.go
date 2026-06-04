package request_logger

import (
	"sync"
	"time"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

const (
	ctxKeyTraceID      = "request_logger_trace_id"
	ctxKeyRequestID    = "request_logger_request_id"
	ctxKeyStartTime    = "request_logger_start_time"
	ctxKeyBidderTimers = "request_logger_bidder_timers"
)

type bidderTimer struct {
	Start time.Time
}

func getStringFromContext(mc *hookstage.ModuleContext, key string) string {
	v, ok := mc.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getTraceID(mc *hookstage.ModuleContext) string {
	return getStringFromContext(mc, ctxKeyTraceID)
}

func getRequestID(mc *hookstage.ModuleContext) string {
	return getStringFromContext(mc, ctxKeyRequestID)
}

func getTraceOrRequestID(mc *hookstage.ModuleContext) string {
	if id := getRequestID(mc); id != "" {
		return id
	}
	return getTraceID(mc)
}

func getStartTime(mc *hookstage.ModuleContext) (time.Time, bool) {
	v, ok := mc.Get(ctxKeyStartTime)
	if !ok {
		return time.Time{}, false
	}
	t, ok := v.(time.Time)
	return t, ok
}

func getBidderTimers(mc *hookstage.ModuleContext) *sync.Map {
	v, ok := mc.Get(ctxKeyBidderTimers)
	if !ok {
		return nil
	}
	m, _ := v.(*sync.Map)
	return m
}

func recordBidderStartTime(mc *hookstage.ModuleContext, bidder string) {
	timers := getBidderTimers(mc)
	if timers == nil {
		return
	}
	timers.Store(bidder, bidderTimer{Start: time.Now()})
}

func getBidderElapsedMs(mc *hookstage.ModuleContext, bidder string) int64 {
	timers := getBidderTimers(mc)
	if timers == nil {
		return 0
	}
	v, ok := timers.Load(bidder)
	if !ok {
		return 0
	}
	t, ok := v.(bidderTimer)
	if !ok {
		return 0
	}
	return time.Since(t.Start).Milliseconds()
}
