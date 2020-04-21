# Otomatik - Automatic HTTPS using Let's Encrypt

We use `otomatic` at [Chamaconekt Kenya](https://github.com/chamaconekt) in our `net.listen` golang programs,
or essentially all of our server applications. This helps us improve and automate network security for all
of our go programs using the ACME protocol.

This protocol gives us TLS certificates, meaning we are getting security from the transport layer. TLS gives us
certain guarantees;

- We want our data to remain private on transit.
- We also want integrity, a guarantee that our data is not modified on transit, and if someone does, we detect it and reject it.
- We want authenticity, a promise that we know we are talking to the right machine and not an impersonator.

With Otomatik, we can add one line to our Go applications to serve securely over TLS, without ever having to touch certificates.

Instead of:

```go
// plaintext HTTP, gross 🤢
http.ListenAndServe(":80", mux)
```

We use Otomatik:

```go
// encrypted HTTPS with HTTP->HTTPS redirects - yay! 🔒😍
otomatik.HTTPS([]string{"example.com"}, mux)
```

This line of code serves our HTTP router `mux` over HTTPS, complete with HTTP->HTTPS redirects.

It obtains and renews the TLS certificates. It staples OCSP responses for greater privacy and security. As long as our domain name points to our server, Otomatik will keep its connections secure.

## Installation

```bash
$ go get github.com/wondenge/otomatik
```

## Credits and License

Otomatik is originally a folk of [CertMagic](https://github.com/caddyserver/otomatik), a project by [Matthew Holt](https://twitter.com/mholt6), who is the author; and other contributors, and now customised and maintained separately for internal use at [Chamaconekt Kenya](https://github.com/chamaconekt). Otomatik is licensed under [Apache 2.0](https://github.com/wondenge/otomatik/blob/master/LICENSE), an open source license.
