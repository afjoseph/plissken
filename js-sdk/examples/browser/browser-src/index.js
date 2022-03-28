// XXX <27-02-22, afjoseph> Put the relative path: webpack will merge everything
// together to one JS file
const lib_opaque = require("../lib/opaque-lib.js");
const hex_encoded_opaque_server_pub_key =
  "f598d3a5880e70fa7ce187bc79122c4895507194ace8bb3ae992d2bfbb7ed63f";
const opaque_server_endpoint = "http://127.0.0.1:3223";
console.log(lib_opaque.ping());

const pub_log_info = (str) => {
  document.getElementById("div-info").innerHTML = `
<div class="alert alert-primary" role="alert">
  Info: ${str}
</div>`;
};

const pub_log_error = (err) => {
  document.getElementById("div-error").innerHTML = `
<div class="alert alert-danger" role="alert">
  Error: ${err}
</div>
`;
};

document.getElementById("btn-register").onclick = async () => {
  const username = document.getElementById("input-username").value;
  const password = document.getElementById("input-password").value;
  if (!username || !password) {
    pub_log_error("Username or password are empty");
    return;
  }

  // Check if server is healthy
  await lib_opaque.check_endpoint_health(opaque_server_endpoint);

  pub_log_info(
    `Running registration with username: ${username} && password: ${password}`
  );
  lib_opaque
    .run_password_reg(
      username,
      password,
      hex_encoded_opaque_server_pub_key,
      opaque_server_endpoint
    )
    .then(() => {
      pub_log_info("Registration done successfully. Try authenticating.");
    })
    .catch((err) => {
      pub_log_error(err);
    });
};

document.getElementById("btn-login").onclick = async () => {
  const username = document.getElementById("input-username").value;
  const password = document.getElementById("input-password").value;
  if (!username || !password) {
    pub_log_error("Username or password are empty");
    return;
  }

  // Check if server is healthy
  await lib_opaque.check_endpoint_health(opaque_server_endpoint);

  pub_log_info("Authenticating...");
  lib_opaque
    .run_password_auth(
      username,
      password,
      hex_encoded_opaque_server_pub_key,
      opaque_server_endpoint
    )
    .then((session_key) => {
      localStorage.setItem("opaque-session-key", session_key);
      pub_log_info(
        `Session key generated and saved in LocalStorage: ${session_key}`
      );
    })
    .catch((err) => {
      pub_log_error(err);
    });
};

document.getElementById("btn-access").onclick = async () => {
  const username = document.getElementById("input-username").value;
  if (!username) {
    pub_log_error("Username is empty");
    return;
  }
  const session_key = localStorage.getItem("opaque-session-key");
  if (!session_key) {
    pub_log_error("Key does not exist. Re-run the registration, please");
    return;
  }

  // Check if server is healthy
  await lib_opaque.check_endpoint_health(opaque_server_endpoint);

  pub_log_info(`Fetching a private resource...`);
  lib_opaque
    .fetch_private_resource(opaque_server_endpoint, username, session_key)
    .then((private_msg) => {
      pub_log_info(`Private message fetched: ${private_msg}`);
    })
    .catch((err) => {
      pub_log_error(err);
    });
};

document.getElementById("div-pubkey").innerHTML = `
<div class="alert alert-primary" role="alert">
  Server's public key: ${hex_encoded_opaque_server_pub_key}
</div>`;

document.getElementById("div-endpoint").innerHTML = `
<div class="alert alert-primary" role="alert">
  Server's endpoint: ${opaque_server_endpoint}
</div>`;
