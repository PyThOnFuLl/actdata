import { api, endpoint, headers, pathParams, request, response, body, Int64 } from "@airtasker/spot";

@api({
	name: "actdata"
})
class Api { }

/**
 * get a list of measurements
 *
 */
@endpoint({
	method: "GET",
	path: "/measurements/"
})
class ListMeasurements {
	@response({ status: 200 }) successResponse(@body body: Array<MeasurementView>) { }
}

/**
 * add new measurement
 *
 */
@endpoint({
	method: "POST",
	path: "/measurements/"
})
class AddMeasurement {
	request(
		@body body: MeasurementView,
	) { }
	@response({ status: 200 }) successResponse() { }
}

/**
 * get session info
 *
 */
@endpoint({
	method: "GET",
	path: "/info/"
})
class GetSessionInfo {
	@response({ status: 200 }) successResponse(@body body: Array<SessionView>) { }
}

/**
 * unix epoch (seconds since 1970 for timestamp or just seconds for time period)
 *
 */
type UnixTime = Int64
type MeasurementView = {
	timestamp: UnixTime,
	heartbeat: number
}
type SessionView = {
	polar_id: Int64
}
