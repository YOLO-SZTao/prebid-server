package request_logger

const defaultMaxPayloadLogSize = 2048

// RequestLoggerConfig defines the module configuration.
type RequestLoggerConfig struct {
	Enabled                     bool `mapstructure:"enabled"`
	LogEntrypoint               bool `mapstructure:"log_entrypoint"`
	LogProcessedAuction         bool `mapstructure:"log_processed_auction"`
	LogBidderRequest            bool `mapstructure:"log_bidder_request"`
	LogRawBidderResponse        bool `mapstructure:"log_raw_bidder_response"`
	LogAllProcessedBidResponses bool `mapstructure:"log_all_processed_bid_responses"`
	LogAuctionResponse          bool `mapstructure:"log_auction_response"`
	LogExitpoint                bool `mapstructure:"log_exitpoint"`
	MaxPayloadLogSize           int  `mapstructure:"max_payload_log_size"`
}

func newConfig() RequestLoggerConfig {
	return RequestLoggerConfig{
		Enabled:                     true,
		LogEntrypoint:               true,
		LogProcessedAuction:         true,
		LogBidderRequest:            true,
		LogRawBidderResponse:        true,
		LogAllProcessedBidResponses: true,
		LogAuctionResponse:          true,
		LogExitpoint:                true,
		MaxPayloadLogSize:           defaultMaxPayloadLogSize,
	}
}

func validateConfig(cfg *RequestLoggerConfig) {
	if cfg.MaxPayloadLogSize <= 0 {
		cfg.MaxPayloadLogSize = defaultMaxPayloadLogSize
	}
}
