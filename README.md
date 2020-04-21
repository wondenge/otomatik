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

## Usage

### Package Overview

#### Certificate authority

This library uses Let's Encrypt by default, but you can use any certificate authority that conforms to the ACME specification. Known/common CAs are provided as consts in the package, for example `LetsEncryptStagingCA` and `LetsEncryptProductionCA`.

#### The `Config` type

The `otomatik.Config` struct is how you can wield the power of this fully armed and operational battle station. However, an empty/uninitialized `Config` is _not_ a valid one! In time, you will learn to use the force of `otomatik.NewDefault()` as I have.

#### Defaults

The default `Config` value is called `otomatik.Default`. Change its fields to suit your needs, then call `otomatik.NewDefault()` when you need a valid `Config` value. In other words, `otomatik.Default` is a template and is not valid for use directly.

You can set the default values easily, for example: `otomatik.Default.Issuer = ...`.

Similarly, to configure ACME-specific defaults, use `otomatik.DefaultACME`.

The high-level functions in this package (`HTTPS()`, `Listen()`, `ManageSync()`, and `ManageAsync()`) use the default config exclusively. This is how most of you will interact with the package. This is suitable when all your certificates are managed the same way. However, if you need to manage certificates differently depending on their name, you will need to make your own cache and configs (keep reading).

#### Providing an email address

Although not strictly required, this is highly recommended best practice. It allows you to receive expiration emails if your certificates are expiring for some reason, and also allows the CA's engineers to potentially get in touch with you if something is wrong. I recommend setting `otomatik.DefaultACME.Email` or always setting the `Email` field of a new `Config` struct.

#### Rate limiting

To avoid firehosing the CA's servers, otomatik has built-in rate limiting. Currently, its default limit is up to 10 transactions (obtain or renew) every 1 minute (sliding window). This can be changed by setting the `RateLimitEvents` and `RateLimitEventsWindow` variables, if desired.

The CA may still enforce their own rate limits, and there's nothing (well, nothing ethical) otomatik can do to bypass them for you.

Additionally, otomatik will retry failed validations with exponential backoff for up to 30 days, with a reasonable maximum interval between attempts (an "attempt" means trying each enabled challenge type once).

### Development and Testing

Note that Let's Encrypt imposes [strict rate limits](https://letsencrypt.org/docs/rate-limits/) at its production endpoint, so using it while developing your application may lock you out for a few days if you aren't careful!

While developing your application and testing it, use [their staging endpoint](https://letsencrypt.org/docs/staging-environment/) which has much higher rate limits. Even then, don't hammer it: but it's much safer for when you're testing. When deploying, though, use their production CA because their staging CA doesn't issue trusted certificates.

To use staging, set `otomatik.DefaultACME.CA = otomatik.LetsEncryptStagingCA` or set `CA` of every `ACMEManager` struct.

### Examples

There are many ways to use this library. We'll start with the highest-level (simplest) and work down (more control).

All these high-level examples use `otomatik.Default` and `otomatik.DefaultACME` for the config and the default cache and storage for serving up certificates.

First, we'll follow best practices and do the following:

```go
// read and agree to your CA's legal documents
otomatik.DefaultACME.Agreed = true

// provide an email address
otomatik.DefaultACME.Email = "you@yours.com"

// use the staging endpoint while we're developing
otomatik.DefaultACME.CA = otomatik.LetsEncryptStagingCA
```

#### Serving HTTP handlers with HTTPS

```go
err := otomatik.HTTPS([]string{"example.com", "www.example.com"}, mux)
if err != nil {
	return err
}
```

This starts HTTP and HTTPS listeners and redirects HTTP to HTTPS!

#### Starting a TLS listener

```go
ln, err := otomatik.Listen([]string{"example.com"})
if err != nil {
	return err
}
```

#### Getting a tls.Config

```go
tlsConfig, err := otomatik.TLS([]string{"example.com"})
if err != nil {
	return err
}
```

#### Advanced use

For more control (particularly, if you need a different way of managing each certificate), you'll make and use a `Cache` and a `Config` like so:

```go
cache := otomatik.NewCache(otomatik.CacheOptions{
	GetConfigForCert: func(cert otomatik.Certificate) (*otomatik.Config, error) {
		// do whatever you need to do to get the right
		// configuration for this certificate; keep in
		// mind that this config value is used as a
		// template, and will be completed with any
		// defaults that are set in the Default config
		return otomatik.Config{
			// ...
		}), nil
	},
	...
})

magic := otomatik.New(cache, otomatik.Config{
	// any customizations you need go here
})

myACME := otomatik.NewACMEManager(magic, ACMEManager{
	CA:     otomatik.LetsEncryptStagingCA,
	Email:  "you@yours.com",
	Agreed: true,
	// plus any other customizations you need
})

magic.Issuer = myACME

// this obtains certificates or renews them if necessary
err := magic.ManageSync([]string{"example.com", "sub.example.com"})
if err != nil {
	return err
}

// to use its certificates and solve the TLS-ALPN challenge,
// you can get a TLS config to use in a TLS listener!
tlsConfig := magic.TLSConfig()

//// OR ////

// if you already have a TLS config you don't want to replace,
// we can simply set its GetCertificate field and append the
// TLS-ALPN challenge protocol to the NextProtos
myTLSConfig.GetCertificate = magic.GetCertificate
myTLSConfig.NextProtos = append(myTLSConfig.NextProtos, tlsalpn01.ACMETLS1Protocol}

// the HTTP challenge has to be handled by your HTTP server;
// if you don't have one, you should have disabled it earlier
// when you made the otomatik.Config
httpMux = myACME.HTTPChallengeHandler(httpMux)
```

Great! This example grants you much more flexibility for advanced programs. However, _the vast majority of you will only use the high-level functions described earlier_, especially since you can still customize them by setting the package-level `Default` config.

### Wildcard certificates

At time of writing (December 2018), Let's Encrypt only issues wildcard certificates with the DNS challenge. You can easily enable the DNS challenge with otomatik for numerous providers (see the relevant section in the docs).

### Behind a load balancer (or in a cluster)

otomatik runs effectively behind load balancers and/or in cluster/fleet environments. In other words, you can have 10 or 1,000 servers all serving the same domain names, all sharing certificates and OCSP staples.

To do so, simply ensure that each instance is using the same Storage. That is the sole criteria for determining whether an instance is part of a cluster.

The default Storage is implemented using the file system, so mounting the same shared folder is sufficient (see [Storage](#storage) for more on that)! If you need an alternate Storage implementation, feel free to use one, provided that all the instances use the _same_ one. :)

See [Storage](#storage) and the associated [pkg.go.dev](https://pkg.go.dev/github.com/caddyserver/otomatik?tab=doc#Storage) for more information!

## Credits and License

Otomatik is originally a folk of [CertMagic](https://github.com/caddyserver/otomatik), a project by [Matthew Holt](https://twitter.com/mholt6), who is the author; and other contributors, and now customised and maintained separately for internal use at [Chamaconekt Kenya](https://github.com/chamaconekt). Otomatik is licensed under [Apache 2.0](https://github.com/wondenge/otomatik/blob/master/LICENSE), an open source license.
