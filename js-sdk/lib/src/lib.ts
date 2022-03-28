import axios from 'axios';
// XXX <28-03-22, afjoseph> Import JS file and then cast it as any so
// it works with Typescript, else you'll get "property does not
// exist on value typeof('blahblah')" errors
import * as _opaque_client from './ext/plissken-bindings/index.js';

const opaque_client = _opaque_client as any;

export function ping(): string {
	return 'pong';
}

/**
/* @throw {Error}
*/
async function check_endpoint_health(endpoint: string): Promise<void> {
	const response = await axios.get(`${endpoint}/health`);
	if (response.status !== 200) {
		throw new Error(`/health route returned bad status code: ${response.status}: ${response.data}`);
	}
}

async function start_password_auth_with_plissken_server(
	endpoint: string,
	oprf_request_result: OprfRequestResult,
): Promise<StartPasswordAuthenticationData> {
	const response = await axios.post(
		`${endpoint}/start_password_authentication`,
		JSON.stringify(oprf_request_result),
	);

	if (response.status !== 200) {
		throw new Error(`/start_password_authentication route returned bad status code: ${response.status}: ${response.data}`);
	}

	return new StartPasswordAuthenticationData(response.data);
}

async function finalize_password_auth_with_plissken_server(
	endpoint: string,
	fin_pass_auth_data: FinalizePasswordAutheticationData,
): Promise<void> {
	const response = await axios.post(
		`${endpoint}/finalize_password_authentication`,
		JSON.stringify(fin_pass_auth_data),
	);
	if (response.status !== 200) {
		throw new Error(`/finalize_password_authentication route returned bad status code: ${response.status}: ${response.data}`);
	}
}

class OprfServerEvaluation {
	elements: string[];
	constructor(js_object: any) {
		if (!('elements' in js_object)) {
			throw new Error(`elements not found in OprfServerEvaluation: ${js_object}`);
		}

		this.elements = js_object.elements;
	}
}

async function start_password_reg_with_plissken_server(
	endpoint: string,
	oprf_request_result: OprfRequestResult,
): Promise<OprfServerEvaluation> {
	const response = await axios.post(
		`${endpoint}/start_password_registration`,
		JSON.stringify(oprf_request_result),
	);
	if (response.status !== 200) {
		throw new Error(`/start_password_registration route returned bad status code: ${response.status}: ${response.data}`);
	}

	return new OprfServerEvaluation(response.data);
}

async function finalize_password_reg_with_plissken_server(
	endpoint: string,
	password_reg_data: PasswordRegistrationData,
) {
	const response = await axios.post(
		`${endpoint}/finalize_password_registration`,
		JSON.stringify(password_reg_data),
	);
	if (response.status != 200) {
		throw `/finalize_password_registration route returned bad status code: ${response.status}: ${response.data}`;
	}
}

export async function run_password_auth(
	apptoken: string,
	username: string,
	password: string,
	opaque_server_pub_key: string,
	opaque_server_endpoint: string,
) {
	console.log(
		`Making password authentication request with password: ${password}`,
	);
	const oprf_request_result = new OprfRequestResult(
		JSON.parse(opaque_client.make_oprf_request(apptoken, username, password)));
	const start_password_auth_data = await start_password_auth_with_plissken_server(
		opaque_server_endpoint,
		oprf_request_result,
	);
	// console.log(
	//   `start_password_auth_data: ${JSON.stringify(
	//     start_password_auth_data,
	//     null,
	//     4,
	//   )}`,
	// );

	const session_token: string = opaque_client.finalize_password_authentication(
		username,
		JSON.stringify(oprf_request_result),
		JSON.stringify(start_password_auth_data),
		opaque_server_pub_key,
	);

	await finalize_password_auth_with_plissken_server(
		opaque_server_endpoint,
		new FinalizePasswordAutheticationData(apptoken, username, session_token),
	);

	console.log(`session_token: ${session_token}`);
	return session_token;
}

class OprfRequestResult {
	apptoken: string;
	username: string;
	inputs: string[];
	blinds: string[];
	eval_req_elements: string[];
	constructor(object: any) {
		if (!('apptoken' in object)) {
			throw new Error(`apptoken not found in OprfRequestResult: ${object}`);
		}

		if (!('username' in object)) {
			throw new Error(`username not found in OprfRequestResult: ${object}`);
		}

		if (!('inputs' in object)) {
			throw new Error(`inputs not found in OprfRequestResult: ${object}`);
		}

		if (!('blinds' in object)) {
			throw new Error(`blinds not found in OprfRequestResult: ${object}`);
		}

		if (!('eval_req_elements' in object)) {
			throw new Error(`eval_req_elements not found in OprfRequestResult: ${object}`);
		}

		this.apptoken = object.apptoken;
		this.username = object.username;
		this.inputs = object.inputs;
		this.blinds = object.blinds;
		this.eval_req_elements = object.eval_req_elements;
	}
}

class StartPasswordAuthenticationData {
	elements: string[];
	envu: string;
	envu_nonce: string;
	rwdu_salt: string;
	auth_nonce: string;
	constructor(object: any) {
		if (!('elements' in object)) {
			throw new Error(`elements not found in StartPasswordAuthenticationData: ${object}`);
		}

		if (!('envu' in object)) {
			throw new Error(`envu not found in StartPasswordAuthenticationData: ${object}`);
		}

		if (!('envu_nonce' in object)) {
			throw new Error(`envu_nonce not found in StartPasswordAuthenticationData: ${object}`);
		}

		if (!('rwdu_salt' in object)) {
			throw new Error(`rwdu_salt not found in StartPasswordAuthenticationData: ${object}`);
		}

		if (!('auth_nonce' in object)) {
			throw new Error(`auth_nonce not found in StartPasswordAuthenticationData: ${object}`);
		}

		this.elements = object.elements;
		this.envu = object.envu;
		this.envu_nonce = object.envu_nonce;
		this.rwdu_salt = object.rwdu_salt;
		this.auth_nonce = object.auth_nonce;
	}
}

class PasswordRegistrationData {
	apptoken: string;
	username: string;
	envu: string;
	envu_nonce: string;
	pubu: string;
	salt: string;
	constructor(object: any) {
		if (!('apptoken' in object)) {
			throw new Error(`apptoken not found in PasswordRegistrationData: ${object}`);
		}

		if (!('username' in object)) {
			throw new Error(`username not found in PasswordRegistrationData: ${object}`);
		}

		if (!('envu' in object)) {
			throw new Error(`envu not found in PasswordRegistrationData: ${object}`);
		}

		if (!('envu_nonce' in object)) {
			throw new Error(`envu_nonce not found in PasswordRegistrationData: ${object}`);
		}

		if (!('pubu' in object)) {
			throw new Error(`pubu not found in PasswordRegistrationData: ${object}`);
		}

		if (!('salt' in object)) {
			throw new Error(`salt not found in PasswordRegistrationData: ${object}`);
		}

		this.apptoken = object.apptoken;
		this.username = object.username;
		this.envu = object.envu;
		this.envu_nonce = object.envu_nonce;
		this.pubu = object.pubu;
		this.salt = object.salt;
	}
}

class FinalizePasswordAutheticationData {
	apptoken: string;
	username: string;
	session_token: string;
	constructor(apptoken: string, username: string, session_token: string) {
		this.apptoken = apptoken;
		this.username = username;
		this.session_token = session_token;
	}
}

export async function run_password_reg(
	apptoken: string,
	username: string,
	password: string,
	opaque_server_pub_key: string,
	opaque_server_endpoint: string,
) {
	console.log(
		`Making password registration request with password: ${password}`,
	);
	const oprf_request_result = new OprfRequestResult(
		JSON.parse(opaque_client.make_oprf_request(apptoken, username, password)));
	const oprf_server_eval = await start_password_reg_with_plissken_server(
		opaque_server_endpoint,
		oprf_request_result,
	);

	const password_reg_data = new PasswordRegistrationData(
		JSON.parse(opaque_client.finalize_password_registration(
			apptoken, username,
			JSON.stringify(oprf_request_result),
			JSON.stringify(oprf_server_eval),
			opaque_server_pub_key,
		)));
	await finalize_password_reg_with_plissken_server(
		opaque_server_endpoint,
		password_reg_data,
	);
	console.log('Password registration successful');
}
