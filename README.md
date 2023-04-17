# Plissken: A Privacy-First Authorization Framework

<p align="center">
  <img src="https://github.com/afjoseph/plissken/blob/main/logo.png" width="300" />
</p>

Plissken provides the backend/frontend code needed to use **Password-Authenticated Key Exchanges** (or PAKE) to perform logins and registrations.

The project streamlines the use of one of the best PAKEs around (the [OPAQUE](https://datatracker.ietf.org/doc/html/draft-krawczyk-cfrg-opaque-06) protocol) for both backend and frontend systems; you can think of this project as a batteries-included PAKE implementation.

The goal of PAKEs is to allow authorization between clients and servers without the servers ever knowing the client's credentials: this means a user's password never needs to leave their device (e.g., browser, phone, IOT device, etc.).

One of the major problems in cybersecurity today that this protocol can solve are **database breaches**. With PAKEs, a company's database will **never** store passwords for a hacker to even consider attacking and breaching. Only a **password proof** of a password exists will ever be stored.

Here's a post by renowned cryptographer Matthew Green [explaining the benefits of PAKE protocols](https://blog.cryptographyengineering.com/2018/10/19/lets-talk-about-pake/) and [another](https://billatnapier.medium.com/eke-its-pake-66c70eddef64) by Professor of cryptography Bill Buchanan explaining the issues with sharing passwords with servers, data breaches and PAKEs in general.

## Overview

### Component Breakdown

This project implements the standardized [OPAQUE](https://datatracker.ietf.org/doc/html/draft-krawczyk-cfrg-opaque-06) protocol for both backend and frontend systems.

The goal of the project is to be plug-and-play: there're both backend and frontend components here to be used with easy configurations for both.

I'll use the same terms as [OAuth2.0]([OAuth2.0](https://www.rfc-editor.org/rfc/rfc6749)) (i.e., tokens, authorization and resource servers) but this project is **not** a substitute for OAuth2.0 (see the FAQ below).


I'll use the names here quite often in the docs and this document. I'll explain the concept again when it's encountered so consider this just a quick reference:

The components for this authorization system are:

- `protocol-lib`: This is the code to understand and work with the OPAQUE protocol, written in Go and located in `./protocol-lib`

- `auth-server`: A server (or part of a server) that implements `protocol-lib`
    - An example of this is `plissken-auth-server`
    - You can use `plissken-auth-server` or just use it as a reference to implement `protocol-lib` in your own project

- `resource-server`: A server (or part of a server) that yields resources given an access token issued by an `auth-server`
    - An example of this is `plissken-example-resource-server`

- `plissken-client`: A client that communicats with an `auth-server` to get tokens so that it can fetch resources from a `resource-server`
    - Two examples here are `plissken-example-nodejs-client` and `plissken-example-webapp-client`, located in `./examples`
    - Both those clients use [GopherJS](https://github.com/gopherjs/gopherjs) to transpile `protocol-lib` to Javascript. You can also do the same by compiling to [WASM]([WebAssembly](https://golangbot.com/webassembly-using-go/)) or even make a library to be used in Android/iOS devices with [Gomobile](https://github.com/golang/mobile)

- `plissken-auth-server`: Plissken authorization server, located in `./auth-server`

- `plissken-js-sdk`: A Javascript implementation of `protocol-lib`
    - Located in `./js-sdk`
    - The Go bindings are in `./auth-server/cmd/js-bindings`
    - See `JS Library Breakdown` section in this document for more info

- `plissken-example-resource-server`: Plissken example resource server, located in `./example-resource-server`

- `plissken-example-nodejs-client`: An example Node.js app that uses `plissken-auth-server` and `plissken-example-resource-server` and acts as a `plissken-client`

- `plissken-example-webapp-client`: An example web app that uses `plissken-auth-server` and `plissken-example-resource-server` and acts as a `plissken-client`


### Directory Breakdown

- `./auth-server/`
    - Example implementation of `plissken-auth-server`
- `./client-examples`
    - `./nodejs`: An example Node.js client that uses `plissken-js-sdk`
    - `./webapp`: An example web app client, written in React, that uses `plissken-js-sdk`
- `./example-resource-server`
    - Example backend usage of `plissken-auth-server`
- `./js-sdk`
    - Houses a Javascript implementation of the protocol (i.e., `plissken-js-sdk`)
- `./protocol-lib`
    - Houses protocol implementation (i.e., `protocol-lib`)
- All `fly.toml` files can be ignored: they are there for the demos. See `Functional Tests -> Against a web app` section of this doc
- `justfile`: Uses [Just](https://github.com/casey/just) command runner to run commands

### JS Library Breakdown

The protocol implementation is written in Go. Rewriting it in Javascript is tedious since you'll have to rewrite every change twice.

What we need will look like this:

       +------------+      +-----------+      +---------------+      +-------------+
       |protocol-lib| ---> |JS Bindings| ---> |plissken-js-sdk| ---> |Any JS Client|
       +------------+      +-----------+      +---------------+      +-------------+

        - protocol-lib:     Plissken protocol library, written in Go
        - JS Bindings:      Go files that provide an easy interface between
                            `plissken-js-sdk` and `protocol-lib`
        - plissken-js-sdk:  Javascript library that calls "Go Bindings"
        - Any JS Client:    Any Javascript code that wants to use Plissken

Here's how the flow of creating a Javascript Plissken implementation goes:

- Write the implementation once in Go
    - This is `protocol-lib`
    - All protocol code changes will be in `./protocol-lib/` only
- Write an interface between the Go and Javascript code in `JS Bindings`
    - This code lives in `./auth-server/cmd/js-bindings`
    - This is the code that will be transpiled with GopherJS to Javascript
    - The generation process occurs with `just generate-js-bindings`
- Write a Javascript library that uses the transpiled Javascript
    - This is `plissken-js-sdk` and it lives in `js-sdk`
    - This won't need to change much over time
- And finally use the code in a Javascript client (i.e., `Any JS Client`)
    - This means just import it from npm as a regular library

## Deployments

- `plissken-auth-server` is deployed in <https://plissken-auth-server.fly.dev>
    - <https://fly.io/apps/plissken-auth-server>
    - Deploy with

        just deploy-auth-server-to-fly


- `plissken-example-resource-server` is deployed in <https://plissken-business-server.fly.dev>
    - <https://fly.io/apps/plissken-business-server>
    - Deploy with

        just deploy-example-resource-server-to-fly


- `plissken-example-webapp-client`: is deployed in <https://plissken-web-demo.fly.dev>
    - <https://fly.io/apps/plissken-web-demo>
    - Deploy with

        just deploy-example-webapp-to-fly


## Dependencies

- Go 1.18
- GopherJS 1.18
    - https://github.com/gopherjs/gopherjs
- Typescript

### Managing Old Go Versions

GopherJS and Go need to have the same major/minor versions.

If you have a way to manage multiple Go installations, use yours.
If not, use this: https://gist.github.com/afjoseph/90218a0cc753219cf5f07aaab28454c5

**Important Note**: If you change your Go version, you **need** to re-download GopherJS.

## Usage

### Generate a JS library of the protocol

By default, this'll be run before any Git commit to the main branch, so you just need to do this if you're developing the library

    just build-js-sdk

## Testing

### Unit Tests

Those are focused on `plissken-auth-server` since the protocol and the main codebase are there.

    just test-plissken-protocol

A very easy way to understand the protocol is to see the unit test for the whole protocol in [./auth-server/main_test.go](https://github.com/afjoseph/plissken/blob/0161debbb075f66116291b3b5db8377ffb8dd3e4/auth-server/main_test.go#L1)

### Functional Tests

The best way to understand the system is to run the different components locally and see how they work

#### Against a Node.js client

This is a non-interactive test that runs a Node.js client against `plissken-auth-server` and `plissken-example-resource-server`

- Run `plissken-auth-server` locally in a terminal window

        just run-plissken-auth-server-local
        // Or debug-plissken-auth-server to run it with delve

- Run `plissken-example-resource-server` locally in another terminal window

        just run-example-resource-server-local

- Compile `plissken-js-sdk` in another terminal window

        just build-js-sdk

- Run `plissken-example-nodejs-client` in the previous terminal window

        just run-example-nodejs-client
        // This will use the compiled plissken-js-sdk

#### Against a web app

This is a non-interactive test that runs an SPA (single-page application) server that hosts a very simple web app as a client.

The client will run the authentication against `plissken-auth-server` and `plissken-example-resource-server`

- Run `plissken-auth-server` locally

        just run-plissken-auth-server-local
        // Or debug-plissken-auth-server to run it with delve

- Run `plissken-example-resource-server` locally

        just run-example-resource-server-local

- Compile `plissken-js-sdk`

        just build-js-sdk

- Run `plissken-example-nodejs-client`

        just run-example-webapp-client
        // This will use the compiled plissken-js-sdk


You can also see web app live in production like this:

- Open <https://plissken-web-demo.fly.dev> with your browser
- Register your username and password
    - This runs the protocol against https://plissken-auth-server.fly.dev
    - You can also open now https://plissken-auth-server.fly.dev with your browser to see the password proofs
- Now login with your new credentials in https://plissken-web-demo.fly.dev
    - After a successful login, `plissken-auth-server` hosted on https://plissken-auth-server.fly.dev would issue an access token to you that you can use
- Put some private resources by pressing `Put private resource`
    - This just communicates with `plissken-example-resource-server` hosted on https://fly.io/apps/plissken-business-server with your new access token
    - `plissken-example-resource-server` would verify your instance by talking with `plissken-auth-server`
- Fetch private resources with `Get private resource`
    - Which does the same as putting a private resource: the client talks to `plissken-example-resource-server` with their access token, and `plissken-example-resource-server` verifies the token with `plissken-auth-server`

## FAQ

### How is this different from OAuth2.0?

[OAuth2.0](https://www.rfc-editor.org/rfc/rfc6749) is an authorization framework that outlines how clients and servers should work together to authorize clients to use resources safely. OAuth2.0 doesn't say anything about **how** a client and server should do authorization. In 99% of cases, backends ask the user for their passwords during authorization and they'll hash it locally.

Plissken is a backend/frontend implementation of a PAKE so users don't have to share passwords with servers. It's been **designed to work in an OAuth2.0 framework** easily since that's the most well-known implementation of client/server authorization on the internet.

To conclude, Plissken **can work with** OAuth2.0 easily.

### Is this Zero-knowledge cryptography?

No. The core author of this repo made a mistake previously by thinking that.

Oblivious Pseudorandom Functions ([OPRFs](https://en.wikipedia.org/wiki/Pseudorandom_function_family#Oblivious_pseudorandom_functions), which this project is based on) are a cryptographic primitive not directly considered part of zero-knowledge cryptography, but they are related cryptographic primitives that can be used in conjunction with zero-knowledge proofs (ZKPs) to achieve certain security properties.

Zero-knowledge cryptography involves cryptographic techniques that allow a prover to prove a statement's truth to a verifier without revealing any additional information beyond the validity of the statement. Think [zk-SNARKs](https://z.cash/technology/zksnarks), [zk-STARKs](https://cointelegraph.com/explained/zk-starks-vs-zk-snarks-explained) and [Bulletproofs](https://blog.pantherprotocol.io/bulletproofs-in-crypto-an-introduction-to-a-non-interactive-zk-proof/).

OPRFs, on the other hand, allow a client and a server to **jointly compute a function over their respective inputs** in a way that [the server remains oblivious to the client's input](https://blog.pantherprotocol.io/bulletproofs-in-crypto-an-introduction-to-a-non-interactive-zk-proof/), and the client learns only the output of the function. In other words, OPRFs enable a two-party computation where one party's input remains hidden from the other party. Think a one-sided Diffie-Hellman.

They **sound alike**: both are used for privacy-preserving applications but they are technically not the same.
