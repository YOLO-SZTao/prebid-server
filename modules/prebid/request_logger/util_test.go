package request_logger

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		limit  int
		expect string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact limit", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"zero limit", "hello", 0, ""},
		{"negative limit", "hello", -1, ""},
		{"empty string", "", 10, ""},
		{"unicode", "你好世界", 3, "..."},
		{"unicode exact", "你好", 2, "你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.limit)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestPreviewBytes(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		limit  int
		expect string
	}{
		{"normal", []byte(`{"id":"test"}`), 100, `{"id":"test"}`},
		{"truncated", []byte(`{"id":"test","imp":[]}`), 10, `{"id":"...`},
		{"empty", []byte{}, 10, ""},
		{"nil", nil, 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := previewBytes(tt.input, tt.limit)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestPreviewJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		limit  int
		expect string
	}{
		{"normal", map[string]string{"key": "value"}, 100, `{"key":"value"}`},
		{"truncated", map[string]int{"a": 1, "b": 2}, 10, `{"a":1,...`},
		{"nil", nil, 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := previewJSON(tt.input, tt.limit)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestGetImpIDs(t *testing.T) {
	req := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{ID: "imp-1"},
				{ID: "imp-2"},
				{ID: ""},
			},
		},
	}
	ids := getImpIDs(req)
	assert.Equal(t, []string{"imp-1", "imp-2"}, ids)
	assert.Nil(t, getImpIDs(nil))
}

func TestGetBidderNames(t *testing.T) {
	impExtWithBidder := func(bidders map[string]json.RawMessage) json.RawMessage {
		ext := map[string]interface{}{
			"prebid": map[string]interface{}{
				"bidder": bidders,
			},
		}
		data, _ := json.Marshal(ext)
		return json.RawMessage(data)
	}

	req := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{ID: "imp-1", Ext: impExtWithBidder(map[string]json.RawMessage{"pubmatic": nil, "rubicon": nil})},
				{ID: "imp-2", Ext: impExtWithBidder(map[string]json.RawMessage{"pubmatic": nil, "appnexus": nil})},
			},
		},
	}

	bidders := getBidderNames(req)
	assert.Equal(t, []string{"appnexus", "pubmatic", "rubicon"}, bidders)
	assert.Nil(t, getBidderNames(nil))
}

func TestGetSource(t *testing.T) {
	assert.Equal(t, "site", getSource(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{}}}))
	assert.Equal(t, "app", getSource(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}}}))
	assert.Equal(t, "dooh", getSource(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{}}}))
	assert.Equal(t, "unknown", getSource(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}))
	assert.Equal(t, "unknown", getSource(nil))
}

func TestGetBidIDs(t *testing.T) {
	resp := &adapters.BidderResponse{
		Bids: []*adapters.TypedBid{
			{Bid: &openrtb2.Bid{ID: "bid-1"}},
			{Bid: &openrtb2.Bid{ID: "bid-2"}},
			{Bid: &openrtb2.Bid{ID: ""}},
		},
	}
	ids := getBidIDs(resp)
	assert.Equal(t, []string{"bid-1", "bid-2"}, ids)
	assert.Nil(t, getBidIDs(nil))
}

func TestCountSeatBids(t *testing.T) {
	assert.Equal(t, 0, countSeatBids(nil))
	assert.Equal(t, 2, countSeatBids(&openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{}, {}},
	}))
}

func TestCountBids(t *testing.T) {
	assert.Equal(t, 0, countBids(nil))
	assert.Equal(t, 3, countBids(&openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{Bid: []openrtb2.Bid{{}, {}}},
			{Bid: []openrtb2.Bid{{}}},
		},
	}))
}

func TestSummarizeExt(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		errB, errD, warnB, warnD, httpB := summarizeExt(nil)
		assert.Nil(t, errB)
		assert.Nil(t, errD)
		assert.Nil(t, warnB)
		assert.Nil(t, warnD)
		assert.Nil(t, httpB)
	})

	t.Run("nil ext", func(t *testing.T) {
		errB, errD, warnB, warnD, httpB := summarizeExt(&openrtb2.BidResponse{})
		assert.Nil(t, errB)
		assert.Nil(t, errD)
		assert.Nil(t, warnB)
		assert.Nil(t, warnD)
		assert.Nil(t, httpB)
	})

	t.Run("malformed ext", func(t *testing.T) {
		errB, errD, warnB, warnD, httpB := summarizeExt(&openrtb2.BidResponse{Ext: json.RawMessage(`not json`)})
		assert.Nil(t, errB)
		assert.Nil(t, errD)
		assert.Nil(t, warnB)
		assert.Nil(t, warnD)
		assert.Nil(t, httpB)
	})

	t.Run("normal ext", func(t *testing.T) {
		ext := openrtb_ext.ExtBidResponse{
			Errors: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
				"rubicon":  {{Code: 1, Message: "timeout"}},
				"pubmatic": {{Code: 2, Message: "error"}},
			},
			Warnings: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
				"appnexus": {{Code: 3, Message: "warning"}},
			},
			Debug: &openrtb_ext.ExtResponseDebug{
				HttpCalls: map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall{
					"pubmatic": {{Uri: "http://example.com"}},
				},
			},
		}
		data, _ := json.Marshal(ext)

		errB, errD, warnB, warnD, httpB := summarizeExt(&openrtb2.BidResponse{Ext: json.RawMessage(data)})
		assert.Equal(t, []string{"pubmatic", "rubicon"}, errB)
		assert.Equal(t, map[string][]map[string]any{
			"pubmatic": {{"code": 2, "message": "error"}},
			"rubicon":  {{"code": 1, "message": "timeout"}},
		}, errD)
		assert.Equal(t, []string{"appnexus"}, warnB)
		assert.Equal(t, map[string][]map[string]any{
			"appnexus": {{"code": 3, "message": "warning"}},
		}, warnD)
		assert.Equal(t, []string{"pubmatic"}, httpB)
	})
}

func TestCollectBidderNamesFromSeatBids(t *testing.T) {
	responses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		"pubmatic": {},
		"rubicon":  {},
	}
	names := collectBidderNamesFromSeatBids(responses)
	assert.Equal(t, []string{"pubmatic", "rubicon"}, names)
}

func TestCountTotalBidsFromSeatBids(t *testing.T) {
	responses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		"pubmatic": {Bids: []*entities.PbsOrtbBid{{}, {}}},
		"rubicon":  {Bids: []*entities.PbsOrtbBid{{}}},
	}
	assert.Equal(t, 3, countTotalBidsFromSeatBids(responses))
}

func TestHeadersToMap(t *testing.T) {
	headers := map[string][]string{
		"Content-Type": {"application/json"},
		"Accept":       {"text/html", "application/json"},
	}
	result := headersToMap(headers)
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "text/html, application/json", result["Accept"])
	assert.Nil(t, headersToMap(nil))
	assert.Nil(t, headersToMap(map[string][]string{}))
}

func TestCollectHttpCallDetails(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		assert.Nil(t, collectHttpCallDetails(nil, 1024))
	})

	t.Run("nil ext", func(t *testing.T) {
		assert.Nil(t, collectHttpCallDetails(&openrtb2.BidResponse{}, 1024))
	})

	t.Run("no debug", func(t *testing.T) {
		ext := openrtb_ext.ExtBidResponse{}
		data, _ := json.Marshal(ext)
		assert.Nil(t, collectHttpCallDetails(&openrtb2.BidResponse{Ext: json.RawMessage(data)}, 1024))
	})

	t.Run("normal httpcalls", func(t *testing.T) {
		ext := openrtb_ext.ExtBidResponse{
			Debug: &openrtb_ext.ExtResponseDebug{
				HttpCalls: map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall{
					"pubmatic": {
						{
							Uri:          "https://pubmatic.com/bid",
							Status:       200,
							RequestBody:  `{"id":"req-1"}`,
							ResponseBody: `{"id":"resp-1"}`,
						},
					},
					"rubicon": {
						{
							Uri:          "https://rubicon.com/bid",
							Status:       204,
							RequestBody:  `{"id":"req-2"}`,
							ResponseBody: "",
						},
					},
				},
			},
		}
		data, _ := json.Marshal(ext)

		result := collectHttpCallDetails(&openrtb2.BidResponse{Ext: json.RawMessage(data)}, 1024)
		assert.Len(t, result, 2)

		pubmatic := result["pubmatic"]
		assert.Len(t, pubmatic, 1)
		assert.Equal(t, "https://pubmatic.com/bid", pubmatic[0]["uri"])
		assert.Equal(t, 200, pubmatic[0]["status"])
		assert.Equal(t, `{"id":"req-1"}`, pubmatic[0]["request_body"])
		assert.Equal(t, `{"id":"resp-1"}`, pubmatic[0]["response_body"])

		rubicon := result["rubicon"]
		assert.Len(t, rubicon, 1)
		assert.Equal(t, 204, rubicon[0]["status"])
	})

	t.Run("truncates bodies", func(t *testing.T) {
		ext := openrtb_ext.ExtBidResponse{
			Debug: &openrtb_ext.ExtResponseDebug{
				HttpCalls: map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall{
					"pubmatic": {{Uri: "https://x.com", Status: 200, RequestBody: "abcdefghij", ResponseBody: "klmnopqrst"}},
				},
			},
		}
		data, _ := json.Marshal(ext)

		result := collectHttpCallDetails(&openrtb2.BidResponse{Ext: json.RawMessage(data)}, 5)
		pubmatic := result["pubmatic"]
		assert.Equal(t, "ab...", pubmatic[0]["request_body"])
		assert.Equal(t, "kl...", pubmatic[0]["response_body"])
	})
}
