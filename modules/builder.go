package modules

import (
	fiftyonedegreesDevicedetection "github.com/prebid/prebid-server/v4/modules/fiftyonedegrees/devicedetection"
	prebidOrtb2blocking "github.com/prebid/prebid-server/v4/modules/prebid/ortb2blocking"
	prebidRequest_logger "github.com/prebid/prebid-server/v4/modules/prebid/request_logger"
	prebidRulesengine "github.com/prebid/prebid-server/v4/modules/prebid/rulesengine"
	scientiamobileWurfl_devicedetection "github.com/prebid/prebid-server/v4/modules/scientiamobile/wurfl_devicedetection"
	scope3Rtd "github.com/prebid/prebid-server/v4/modules/scope3/rtd"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"fiftyonedegrees": {
			"devicedetection": fiftyonedegreesDevicedetection.Builder,
		},
		"prebid": {
			"ortb2blocking":  prebidOrtb2blocking.Builder,
			"request_logger": prebidRequest_logger.Builder,
			"rulesengine":    prebidRulesengine.Builder,
		},
		"scientiamobile": {
			"wurfl_devicedetection": scientiamobileWurfl_devicedetection.Builder,
		},
		"scope3": {
			"rtd": scope3Rtd.Builder,
		},
	}
}
