import axios from 'axios';
import * as plissken_js_sdk from 'plissken-js-sdk';

async function main() {
  const apptoken = 'my-app-token';
  const username = 'truebeef';
  const password = 'bunnyfoofoo';
  const hex_encoded_plissken_server_pub_key
    = 'f598d3a5880e70fa7ce187bc79122c4895507194ace8bb3ae992d2bfbb7ed63f';
  const plissken_auth_server_endpoint = 'http://127.0.0.1:3223';
  const resource_server_endpoint = 'http://127.0.0.1:3224';
  try {
    await plissken_js_sdk.run_password_reg(
      apptoken,
      username,
      password,
      hex_encoded_plissken_server_pub_key,
      plissken_auth_server_endpoint,
    );

    const session_token: string = await plissken_js_sdk.run_password_auth(
      apptoken,
      username,
      password,
      hex_encoded_plissken_server_pub_key,
      plissken_auth_server_endpoint,
    );

    // const session_token = 'b1066c68613e17cbe7e3c2f78ec51a44a67ef94a8748366f1dde4424f0e4da69';

    let response = await axios.put(
      `${resource_server_endpoint}/put-resource`,
      null,
      {
        params: {
          session_token,
          username,
          ts: Date.now().toString(),
        },
      },
    );
    if (response.status !== 200) {
      throw new Error('Failed to put value');
    }

    console.log('Getting the same value from the resource server');
    response = await axios.get(
      `${resource_server_endpoint}/get-resource`,
      {
        params: {
          session_token,
          username,
        },
      },
    );
    if (response.status !== 200) {
      throw new Error('Failed to put value');
    }

    let d = new Date();
    d.setTime(response.data);
    console.log(`Successfully got last time I cut my hair: ${d.toUTCString()}`);
  } catch (error: unknown) {
    console.error(`From plissken-js-sdk: ${error}`);
  }
}

await main();
