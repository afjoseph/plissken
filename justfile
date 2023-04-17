set export := true

protocol_lib_path := justfile_directory() + "/protocol-lib"
default_js_out_path := justfile_directory() + "/js-sdk/src/ext/plissken-bindings/lib.js"
auth_server_path := justfile_directory() + "/auth-server/"
auth_server_local_config_path := auth_server_path + '/configs/local.yml'
js_bindings_path := auth_server_path + "/cmd/js-bindings"
example_client_node_path := justfile_directory() + "/client-examples/nodejs/"
js_sdk_path := justfile_directory() + "/js-sdk/"
example_resource_server_path := justfile_directory() + "/example-resource-server/"
example_resource_server_local_config_path := example_resource_server_path + '/configs/local.yml'
example_webapp_path := justfile_directory() + '/client-examples/webapp/'
example_webapp_frontend_path := example_webapp_path + '/frontend/'
git_commit_hash := `git rev-parse HEAD`
production_plissken_auth_server_url := "https://plissken-auth-server.fly.dev"
production_plissken_auth_server_pubkey := "bbf737bfb3417bba7b14d7690679ad3a6e880ebf9a653b5ef3792a888d44fb22"
production_plissken_resource_server_url := "https://plissken-business-server.fly.dev"
local_plissken_auth_server_url := "http://127.0.0.1:3223"
local_plissken_auth_server_pubkey := "f598d3a5880e70fa7ce187bc79122c4895507194ace8bb3ae992d2bfbb7ed63f"
local_plissken_resource_server_url := "http://127.0.0.1:3224"

# plissken-protocol
# ---------------

test-plissken-protocol:
    cd {{ protocol_lib_path }} && go test -failfast ./...

# plissken-auth-server
# ------------------

run-plissken-auth-server-local:
    cd {{ auth_server_path }} && go run . -config-path={{ auth_server_local_config_path }}

deploy-auth-server-to-fly:
    (cd {{ auth_server_path }} && \
      go mod vendor && \
      flyctl deploy --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})

# example-resource-server
# --------------------------

run-example-resource-server-local:
    cd {{ example_resource_server_path }} && go run . -config-path={{ example_resource_server_local_config_path }}

deploy-example-resource-server-to-fly:
    (cd {{ example_resource_server_path }} && flyctl deploy \
      --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})

# plissken-js-sdk
# ----------------

generate-js-bindings:
    cd {{ js_bindings_path }} && gopherjs build \
      --minify --output {{ default_js_out_path }}
    @# Possibly add this later: uglifyjs out/plissken.js --mangle --compress \
    @# sequences=true,dead_code=true,conditionals=true,booleans=true,unused=true,if_return=true,join_vars=true,drop_console=true \
    @# --output out/plissken.min.js

build-js-sdk: generate-js-bindings
    (cd {{ js_sdk_path }} && npm install && npm run build)

# plissken-example-nodejs-client
# -------------------------------

run-example-nodejs-client:
    (cd {{ example_client_node_path }} && npm install && npm run run)

# plissken-example-webapp-client
# -------------------------------
# XXX <17-04-2022, afjoseph> We're running this with a shebang since, in Just,
# each line is a separate shell command. This means exported variables won't be
# reflected in other commands. If we run it in our own explicitly-declared
# shebang, we can export variables and have it reflected in other tasks without
# an issue: https://github.com/casey/just/issues/282#issuecomment-349078653.
#
# Furthermore, any exported variables prefixed with REACT_APP will be available
# in a React app using "process.env".

run-example-webapp-client:
    #!/usr/bin/env bash
    set -euxo pipefail
    export REACT_APP_PLISSKEN_AUTH_SERVER_URL={{ local_plissken_auth_server_url }}
    export REACT_APP_PLISSKEN_RESOURCE_SERVER_URL={{ local_plissken_resource_server_url }}
    export REACT_APP_PLISSKEN_AUTH_SERVER_PUBKEY={{ local_plissken_auth_server_pubkey }}
    (cd {{ example_webapp_frontend_path }} && npm install && npm run start)

deploy-example-webapp-to-fly: build-js-sdk
    #!/usr/bin/env bash
    set -euxo pipefail
    export REACT_APP_PLISSKEN_AUTH_SERVER_URL={{ production_plissken_auth_server_url }}
    export REACT_APP_PLISSKEN_RESOURCE_SERVER_URL={{ production_plissken_resource_server_url }}
    export REACT_APP_PLISSKEN_AUTH_SERVER_PUBKEY={{ production_plissken_auth_server_pubkey }}
    echo "Building frontend..."
    (cd {{ example_webapp_frontend_path }} && npm install && npm run build)
    echo "Building Deploying to Fly..."
    (cd {{ example_webapp_path }} && flyctl deploy \
      --build-arg GIT_COMMIT_HASH={{ git_commit_hash }})
