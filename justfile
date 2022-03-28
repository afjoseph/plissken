set export := true

chrome_executable := env_var_or_default("CHROME_EXECUTABLE", "/usr/bin/env google-chrome-stable")
default_gopherjs_out_path := justfile_directory() + "/js-sdk/lib/src/ext/plissken-bindings/lib.js"
auth_server_path := justfile_directory() + "/auth-server/"
auth_server_test_config_path := auth_server_path + '/configs/test.yml'
plissken_protocol_path := justfile_directory() + "/protocol"
gopherjs_bindings_path := auth_server_path + "/cmd/gopherjs-bindings"
example_client_node_path := justfile_directory() + "/js-sdk/examples/nodejs/"
js_sdk_path := justfile_directory() + "/js-sdk/lib/"
example_business_server_path := justfile_directory() + "/example-business-server/"
example_business_server_test_config_path := example_business_server_path + '/configs/test.yml'
web_demo_path := justfile_directory() + '/web-demo/'
web_demo_frontend_path := web_demo_path + '/frontend/'
web_demo_spa_server_path := web_demo_path + '/spa-server/'
git_commit_hash := `git rev-parse HEAD`
production_plissken_auth_server_url := "https://plissken-auth-server.fly.dev"
production_plissken_auth_server_pubkey := "bbf737bfb3417bba7b14d7690679ad3a6e880ebf9a653b5ef3792a888d44fb22"
production_plissken_business_server_url := "https://plissken-business-server.fly.dev"
debug_plissken_auth_server_url := "http://127.0.0.1:3223"
debug_plissken_auth_server_pubkey := "f598d3a5880e70fa7ce187bc79122c4895507194ace8bb3ae992d2bfbb7ed63f"
debug_plissken_business_server_url := "http://127.0.0.1:3224"

# plissken-auth-server
# ----------

test-plissken-auth-server:
    cd {{ auth_server_path }} && go test -failfast ./...

run-plissken-auth-server-debug:
    cd {{ auth_server_path }} && go run . -config-path={{ auth_server_test_config_path }}

generate-gopherjs-bindings OUT_PATH=default_gopherjs_out_path:
    cd {{ gopherjs_bindings_path }} && gopherjs build \
      --minify --output {{ OUT_PATH }}
    @# Possibly add this later: uglifyjs out/plissken.js --mangle --compress \
    @# sequences=true,dead_code=true,conditionals=true,booleans=true,unused=true,if_return=true,join_vars=true,drop_console=true \
    @# --output out/plissken.min.js

deploy-auth-server-to-fly:
    (cd {{ auth_server_path }} && \
      flyctl deploy --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})

# example-business-server
# ----------

run-example-business-server_debug:
    cd {{ example_business_server_path }} && go run . -config-path={{ example_business_server_test_config_path }}

deploy-business-server-to-fly:
    (cd {{ example_business_server_path }} && flyctl deploy \
      --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})

# plissken-js-sdk
# ----------------

build-js-sdk:
    (cd {{ js_sdk_path }} && npm run build)

run-example-client-node:
    (cd {{ example_client_node_path }} && npm run run)

# plissken-web-demo
# ----------------
# XXX <17-04-2022, afjoseph> We're running this with a shebang since, in Just,
# each line is a separate shell command. This means exported variables won't be
# reflected in other commands. If we run it in our own explicitly-declared
# shebang, we can export variables and have it reflected in other tasks without
# an issue: https://github.com/casey/just/issues/282#issuecomment-349078653.
#
# Furthermore, any exported variables prefixed with REACT_APP will be available
# in a React app using "process.env".

run-web-demo-debug:
    #!/usr/bin/env bash
    set -euxo pipefail
    export REACT_APP_PLISSKEN_AUTH_SERVER_URL={{ debug_plissken_auth_server_url }}
    export REACT_APP_PLISSKEN_BUSINESS_SERVER_URL={{ debug_plissken_business_server_url }}
    export REACT_APP_PLISSKEN_AUTH_SERVER_PUBKEY={{ debug_plissken_auth_server_pubkey }}
    (cd {{ web_demo_frontend_path }} && npm install && npm run start)

deploy-web-demo-to-fly: generate-gopherjs-bindings build-js-sdk
    #!/usr/bin/env bash
    set -euxo pipefail
    export REACT_APP_PLISSKEN_AUTH_SERVER_URL={{ production_plissken_auth_server_url }}
    export REACT_APP_PLISSKEN_BUSINESS_SERVER_URL={{ production_plissken_business_server_url }}
    export REACT_APP_PLISSKEN_AUTH_SERVER_PUBKEY={{ production_plissken_auth_server_pubkey }}
    echo "Building frontend..."
    (cd {{ web_demo_frontend_path }} && npm install && npm run build)
    echo "Building Deploying to Fly..."
    (cd {{ web_demo_path }} && flyctl deploy \
      --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})
