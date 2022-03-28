import axios from "axios";
import * as plissken_js_sdk from "plissken-js-sdk";
import React from "react";
import "./login_form.css";

const default_app_token = "my-app-token";
const default_username = "";
const default_password = "";
const hex_encoded_plissken_server_pub_key =
  process.env.REACT_APP_PLISSKEN_AUTH_SERVER_PUBKEY;
const plissken_server_endpoint = process.env.REACT_APP_PLISSKEN_AUTH_SERVER_URL;
const business_server_endpoint =
  process.env.REACT_APP_PLISSKEN_BUSINESS_SERVER_URL;

class MyForm extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      username: default_username,
      password: default_password,
      session_token:
        "0000000000000000000000000000000000000000000000000000000000000000",
      info_msg: "",
      error_msg: "",
    };

    this.handleLoginBtn = this.handleLoginBtn.bind(this);
    this.handleRegisterBtn = this.handleRegisterBtn.bind(this);
    this.handlePutResourceBtn = this.handlePutResourceBtn.bind(this);
    this.handleGetResourceBtn = this.handleGetResourceBtn.bind(this);
  }

  async handleLoginBtn() {
    try {
      const session_token = await plissken_js_sdk.run_password_auth(
        default_app_token,
        this.state.username,
        this.state.password,
        hex_encoded_plissken_server_pub_key,
        plissken_server_endpoint
      );
      this.setState({ session_token });
      this.setState({ info_msg: "Successfully logged in", error_msg: "" });
    } catch (error) {
      console.error(`From plissken-js-sdk: ${error}`);
      if (axios.isAxiosError(error)) {
        let err = `Failed to register: ${error}`;
        if (error.response && error.response.data) {
          err += `: ${error.response.data}`;
        }
        this.setState({
          error_msg: err,
          info_msg: "",
        });
      } else {
        this.setState({
          error_msg: `Failed to login: ${error}`,
          info_msg: "",
        });
      }
    }
  }

  async handleRegisterBtn() {
    try {
      await plissken_js_sdk.run_password_reg(
        default_app_token,
        this.state.username,
        this.state.password,
        hex_encoded_plissken_server_pub_key,
        plissken_server_endpoint
      );
      this.setState({ info_msg: "Successfully registered", error_msg: "" });
    } catch (error) {
      console.error(`From plissken-js-sdk: ${error}`);
      if (axios.isAxiosError(error)) {
        let err = `Failed to register: ${error}`;
        if (error.response && error.response.data) {
          err += `: ${error.response.data}`;
        }
        this.setState({
          error_msg: err,
          info_msg: "",
        });
      } else {
        this.setState({
          error_msg: `Failed to register: ${error}`,
          info_msg: "",
        });
      }
    }
  }

  async handlePutResourceBtn() {
    try {
      let response = await axios.put(
        `${business_server_endpoint}/put-resource`,
        null,
        {
          params: {
            session_token: this.state.session_token,
            username: this.state.username,
            ts: Date.now().toString(),
          },
        }
      );
      if (response.status !== 200) {
        throw new Error("Failed to put value");
      }
      this.setState({
        info_msg: `Updated last time I cut my hair`,
        error_msg: "",
      });
    } catch (error) {
      console.error(`From plissken-js-sdk: ${error}`);
      if (axios.isAxiosError(error)) {
        let err = `Failed to put resource: ${error}`;
        if (error.response && error.response.data) {
          err += `: ${error.response.data}`;
        }
        this.setState({
          error_msg: err,
          info_msg: "",
        });
      } else {
        this.setState({
          error_msg: `Failed: ${error}`,
          info_msg: "",
        });
      }
    }
  }

  async handleGetResourceBtn() {
    try {
      let response = await axios.get(
        `${business_server_endpoint}/get-resource`,
        {
          params: {
            session_token: this.state.session_token,
            username: this.state.username,
          },
        }
      );
      if (response.status !== 200) {
        throw new Error("Failed to put value");
      }

      let d = new Date();
      d.setTime(response.data);
      this.setState({
        info_msg: `Successfully got last time I cut my hair: ${d.toUTCString()}`,
        error_msg: "",
      });
      // TODO <15-04-2022, afjoseph> Display it somewhere nicer
    } catch (error) {
      console.error(`From plissken-js-sdk: ${error}`);
      if (axios.isAxiosError(error)) {
        let err = `Failed to get resource: ${error}`;
        if (error.response && error.response.data) {
          err += `: ${error.response.data}`;
        }
        this.setState({
          error_msg: err,
          info_msg: "",
        });
      } else {
        this.setState({
          error_msg: `Failed: ${error}`,
          info_msg: "",
        });
      }
    }
  }

  render() {
    return (
      <div class="m-5">
        <form class="login-form">
          <div class="form-group mx-5">
            <label for="username">Username</label>
            <input
              type="username"
              class="form-control"
              id="input-username"
              placeholder="Enter Username"
              value={this.state.username}
              onChange={(ev) => {
                this.setState({ username: ev.target.value });
              }}
            />
            <small class="form-text text-muted">
              We'll never share your username with anyone else.
            </small>
          </div>

          <div class="form-group mb-3 mx-5">
            <label for="password">Password</label>
            <input
              type="password"
              class="form-control"
              id="input-password"
              placeholder="Password"
              value={this.state.password}
              onChange={(ev) => {
                this.setState({ password: ev.target.value });
              }}
            />
          </div>

          <div class="btns">
            <div class="m-2">
              <button
                margin="50px"
                type="button"
                id="btn-register"
                class="btn btn-primary"
                onClick={this.handleRegisterBtn}
              >
                Register
              </button>
            </div>
            <div class="m-2">
              <button
                type="button"
                id="btn-login"
                class="btn btn-primary"
                onClick={this.handleLoginBtn}
              >
                Login
              </button>
            </div>
            <div class="m-2">
              <button
                type="button"
                class="btn btn-primary"
                onClick={this.handlePutResourceBtn}
              >
                Put private resource
              </button>
            </div>
            <div class="m-2">
              <button
                type="button"
                class="btn btn-primary"
                onClick={this.handleGetResourceBtn}
              >
                Get private resource
              </button>
            </div>
          </div>
        </form>

        <div class="labels m-5 d-grid gap-3">
          <div>
            <label>session-token:</label>
            <br />
            <label>{this.state.session_token}</label>
          </div>
          {this.state.info_msg && (
            <div>
              <h3 class="error"> {this.state.info_msg} </h3>
            </div>
          )}
          {this.state.error_msg && (
            <div>
              <h3 classFor="error"> {this.state.error_msg} </h3>
            </div>
          )}
        </div>
      </div>
    );
  }
}

export default MyForm;
