import { api, endpoint, headers, pathParams, request, response, body, Int64, queryParams } from "@airtasker/spot";

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
	@request
	request(
		@body body: MeasurementView,
	) { }
	@response({ status: 200 }) successResponse() { }
}

/**
 * start a new session or continue an old one with code from polar oauth
 * and receive auth token
 */
@endpoint({
	method: "GET",
	path: "/oauth2_callback/"
})
class Oauth {
	@request
	request( @queryParams qp: {code: string}){}
	@response({ status: 200 }) successResponse(@body body: string) { }
}

/**
 * get session info (polar ID needed e.g. for /proxy/users/$id)
 *
 */
@endpoint({
	method: "GET",
	path: "/info/"
})
class GetSessionInfo {
	@response({ status: 200 }) successResponse(@body body: SessionView) { }
}

/**
 * proxy to accesslink
 * e.g. /proxy/users/123 -> https://www.polaraccesslink.com/v3/users/123
 */
@endpoint({
	method: "HEAD",
	path: "/proxy/"
})
class Proxy {
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
	polar_id: Int64,
	session_id: Int64
}
