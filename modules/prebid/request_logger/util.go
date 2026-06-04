package request_logger

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func truncateString(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-3]) + "..."
}

func previewBytes(b []byte, limit int) string {
	if len(b) == 0 {
		return ""
	}
	return truncateString(string(b), limit)
}

func previewJSON(v any, limit int) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return truncateString(string(data), limit)
}

func getImpIDs(req *openrtb_ext.RequestWrapper) []string {
	if req == nil {
		return nil
	}
	imps := req.GetImp()
	ids := make([]string, 0, len(imps))
	for _, imp := range imps {
		if imp != nil && imp.ID != "" {
			ids = append(ids, imp.ID)
		}
	}
	return ids
}

func getBidderNames(req *openrtb_ext.RequestWrapper) []string {
	if req == nil {
		return nil
	}
	imps := req.GetImp()
	seen := make(map[string]bool)
	var bidders []string
	for _, imp := range imps {
		impExt, err := imp.GetImpExt()
		if err != nil || impExt == nil {
			continue
		}
		prebid := impExt.GetPrebid()
		if prebid == nil || prebid.Bidder == nil {
			continue
		}
		for bidder := range prebid.Bidder {
			if !seen[bidder] {
				seen[bidder] = true
				bidders = append(bidders, bidder)
			}
		}
	}
	sort.Strings(bidders)
	return bidders
}

func getSource(req *openrtb_ext.RequestWrapper) string {
	if req == nil {
		return "unknown"
	}
	if req.Site != nil {
		return "site"
	}
	if req.App != nil {
		return "app"
	}
	if req.DOOH != nil {
		return "dooh"
	}
	return "unknown"
}

func getBidIDs(resp *adapters.BidderResponse) []string {
	if resp == nil {
		return nil
	}
	ids := make([]string, 0, len(resp.Bids))
	for _, bid := range resp.Bids {
		if bid != nil && bid.Bid != nil && bid.Bid.ID != "" {
			ids = append(ids, bid.Bid.ID)
		}
	}
	return ids
}

func getBidImpIDs(resp *adapters.BidderResponse) []string {
	if resp == nil {
		return nil
	}
	ids := make([]string, 0, len(resp.Bids))
	for _, bid := range resp.Bids {
		if bid != nil && bid.Bid != nil && bid.Bid.ImpID != "" {
			ids = append(ids, bid.Bid.ImpID)
		}
	}
	return ids
}

func countSeatBids(resp *openrtb2.BidResponse) int {
	if resp == nil {
		return 0
	}
	return len(resp.SeatBid)
}

func countBids(resp *openrtb2.BidResponse) int {
	if resp == nil {
		return 0
	}
	n := 0
	for _, seat := range resp.SeatBid {
		n += len(seat.Bid)
	}
	return n
}

func summarizeExt(resp *openrtb2.BidResponse) (errorBidders []string, errorDetails map[string][]map[string]any, warningBidders []string, warningDetails map[string][]map[string]any, httpcallBidders []string) {
	if resp == nil || resp.Ext == nil {
		return nil, nil, nil, nil, nil
	}

	var ext openrtb_ext.ExtBidResponse
	if err := json.Unmarshal(resp.Ext, &ext); err != nil {
		return nil, nil, nil, nil, nil
	}

	errorDetails = make(map[string][]map[string]any)
	for bidder, msgs := range ext.Errors {
		bidderStr := string(bidder)
		errorBidders = append(errorBidders, bidderStr)
		for _, m := range msgs {
			errorDetails[bidderStr] = append(errorDetails[bidderStr], map[string]any{
				"code":    m.Code,
				"message": m.Message,
			})
		}
	}
	sort.Strings(errorBidders)

	warningDetails = make(map[string][]map[string]any)
	for bidder, msgs := range ext.Warnings {
		bidderStr := string(bidder)
		warningBidders = append(warningBidders, bidderStr)
		for _, m := range msgs {
			warningDetails[bidderStr] = append(warningDetails[bidderStr], map[string]any{
				"code":    m.Code,
				"message": m.Message,
			})
		}
	}
	sort.Strings(warningBidders)

	if ext.Debug != nil {
		for bidder := range ext.Debug.HttpCalls {
			httpcallBidders = append(httpcallBidders, string(bidder))
		}
		sort.Strings(httpcallBidders)
	}

	return
}

func collectBidderNamesFromSeatBids(responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) []string {
	names := make([]string, 0, len(responses))
	for name := range responses {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}

func countTotalBidsFromSeatBids(responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) int {
	n := 0
	for _, seat := range responses {
		n += len(seat.Bids)
	}
	return n
}

func headersToMap(headers map[string][]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		result[k] = strings.Join(v, ", ")
	}
	return result
}

func collectHttpCallDetails(resp *openrtb2.BidResponse, maxSize int) map[string][]map[string]any {
	if resp == nil || resp.Ext == nil {
		return nil
	}

	var ext openrtb_ext.ExtBidResponse
	if err := json.Unmarshal(resp.Ext, &ext); err != nil || ext.Debug == nil {
		return nil
	}

	result := make(map[string][]map[string]any)
	for bidder, calls := range ext.Debug.HttpCalls {
		bidderStr := string(bidder)
		for _, c := range calls {
			if c == nil {
				continue
			}
			result[bidderStr] = append(result[bidderStr], map[string]any{
				"uri":           c.Uri,
				"status":        c.Status,
				"request_body":  truncateString(c.RequestBody, maxSize),
				"response_body": truncateString(c.ResponseBody, maxSize),
			})
		}
	}
	return result
}

