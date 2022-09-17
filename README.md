# Plissken: Privacy-First, Zero-Knowledge Password Authentication Suite

<p align="center">
  <img src="https://github.com/afjoseph/plissken/blob/main/logo.png" width="300" />
</p>

Plissken provides the backend/frontend code needed to use a new concept in cryptography called **Password-Authenticated Key Exchanges** (or PAKE) to perform login/registrations for a website so that your credentials never leave your device (e.g., browser, phone, IOT device, etc.) but a website can still authenticate you correctly.

One of the major problems in cybersecurity today that this protocol can solve are database breaches. With this protocol, a company's database will **never** store any passwords for a hacker to even consider attacking and breaching. Your password will never leave your device. Only a **proof** that your password exists will ever be stored.

## Technical Jargon

Plissken does not invent a new concept in cryptography, but simply implements the standardized [OPAQUE](https://datatracker.ietf.org/doc/html/draft-krawczyk-cfrg-opaque-06) protocol for both backend and frontend systems. Plissken uses Go to achieve this since it's easy to create shared libraries for use in a backend system, transpile to JS (with [GopherJS](https://datatracker.ietf.org/doc/html/draft-krawczyk-cfrg-opaque-06)), build directly to [WebAssembly](https://golangbot.com/webassembly-using-go/), use in an IOT device with [TinyGo](https://tinygo.org/), or build to shared libraries that both Android and iOS devices can understand with [Gomobile](https://github.com/golang/mobile).

Here's a post by renowned cryptographer Matthew Green [explaining the benefits of PAKE protocols](https://blog.cryptographyengineering.com/2018/10/19/lets-talk-about-pake/) and [another](https://billatnapier.medium.com/eke-its-pake-66c70eddef64) by Professor of cryptography Bill Buchanan.

## Demos

See demos here:

- Auth server: https://plissken-auth-server.fly.dev
  - this is where all the password proofs will be stored. This can be a different server than the resources server
- Business server: https://plissken-business-server.fly.dev
  - This is the server the user would access after authentication.
- Web demo: https://plissken-web-demo.fly.dev
  - Just a web demo that ties both the above concepts together

To summarize the demo:

- A user opens https://plissken-web-demo.fly.dev with their browser
- They register there with https://plissken-auth-server.fly.dev. Their browser basically generates password proofs and runs the OPAQUE protocol against the Auth Server.
- After a successful registration, the user would login to the Auth Server
- Auth Server would issue an ephemeral token to them that they can use with the Business Server
- Now, the user can use the Business Server using their issued ephemeral token
- When the Business Server sees a user request to access a resource, they would query the Auth Server (through a server-to-server request) to check if the user has authenticated successfully
- If they did, the Business Server can return the requested resource to the user.

## Dependencies

- Go 1.17
- GopherJS 1.17

## Usage

- Run a plissken-auth-server locally

    just run-plissken-auth-server
    // Or debug-plissken-auth-server to run it with delve

- Build plissken-js-sdk

    just build-js-sdk
    // Output is in ./js-sdk/lib/dist
    // Other projects can include this with `npm install plissken-js-sdk`

- Run an example nodejs app with plissken-js-sdk locally with demo defaults

    // You need to have a plissken-auth-server running locally on
    // http://localhost:3223 using the key in ./server/testdata/privkey.
    // If you run `just run-plissken-auth-server`, things will work just fine
    just generate-gopherjs-bindings && \
      just build-js-sdk && \
      just run-example-client-node
