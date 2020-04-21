# Otomatik - Automatic HTTPS using Let's Encrypt

With Otomatik, you can add one line to your Go application to serve securely over TLS, without ever having to touch certificates.

Instead of:

```go
// plaintext HTTP, gross 🤢
http.ListenAndServe(":80", mux)
```

Use Otomatik:

```go
// encrypted HTTPS with HTTP->HTTPS redirects - yay! 🔒😍
otomatik.HTTPS([]string{"example.com"}, mux)
```

That line of code will serve your HTTP router `mux` over HTTPS, complete with HTTP->HTTPS redirects. It obtains and renews the TLS certificates. It staples OCSP responses for greater privacy and security. As long as your domain name points to your server, Otomatik will keep its connections secure.



## Features

- Fully automated certificate management including issuance and renewal
- One-liner, fully managed HTTPS servers
- Full control over almost every aspect of the system
- HTTP->HTTPS redirects
- Solves all 3 ACME challenges: HTTP, TLS-ALPN, and DNS
- Most robust error handling of _any_ ACME client - Challenges are randomized to avoid accidental dependence - Challenges are rotated to overcome certain network blockages - Robust retries for up to 30 days - Exponential backoff with carefully-tuned intervals - Retries with optional test/staging CA endpoint instead of production, to avoid rate limits
- Over 50 DNS providers work out-of-the-box (powered by [lego](https://github.com/go-acme/lego)!)
- Written in Go, a language with memory-safety guarantees
- Pluggable storage implementations (default: file system)
- Wildcard certificates
- Automatic OCSP stapling ([done right](https://gist.github.com/sleevi/5efe9ef98961ecfb4da8#gistcomment-2336055)) [keeps your sites online!](https://twitter.com/caddyserver/status/1234874273724084226) - Will [automatically attempt](https://twitter.com/mholt6/status/1235577699541762048) to replace [revoked certificates](https://community.letsencrypt.org/t/2020-02-29-caa-rechecking-bug/114591/3?u=mholt)! - Staples stored to disk in case of responder outages
- Distributed solving of all challenges (works behind load balancers) - Highly efficient, coordinated management in a fleet - Active locking - Smart queueing
- Supports "on-demand" issuance of certificates (during TLS handshakes!) - Caddy / otomatik pioneered this technology - Custom decision functions to regulate and throttle on-demand behavior
- Optional event hooks for observation
- Works with any certificate authority (CA) compliant with the ACME specification
- Certificate revocation (please, only if private key is compromised)
- Must-Staple (optional; not default)
- Cross-platform support! Mac, Windows, Linux, BSD, Android...
- Scales to hundreds of thousands of names/certificates per instance
- Use in conjunction with your own certificates

## Requirements

1. Public DNS name(s) you control
2. Server reachable from public Internet
   - Or use the DNS challenge to waive this requirement
3. Control over port 80 (HTTP) and/or 443 (HTTPS)
   - Or they can be forwarded to other ports you control
   - Or use the DNS challenge to waive this requirement
   - (This is a requirement of the ACME protocol, not a library limitation)
4. Persistent storage
   - Typically the local file system (default)
   - Other integrations available/possible

**_Before using this library, your domain names MUST be pointed (A/AAAA records) at your server (unless you use the DNS challenge)!_**

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

## The ACME Challenges

This section describes how to solve the ACME challenges. Challenges are how you demonstrate to the certificate authority some control over your domain name, thus authorizing them to grant you a certificate for that name. [The great innovation of ACME](https://www.dotconferences.com/2016/10/matthew-holt-go-with-acme) is that verification by CAs can now be automated, rather than having to click links in emails (who ever thought that was a good idea??).

If you're using the high-level convenience functions like `HTTPS()`, `Listen()`, or `TLS()`, the HTTP and/or TLS-ALPN challenges are solved for you because they also start listeners. However, if you're making a `Config` and you start your own server manually, you'll need to be sure the ACME challenges can be solved so certificates can be renewed.

The HTTP and TLS-ALPN challenges are the defaults because they don't require configuration from you, but they require that your server is accessible from external IPs on low ports. If that is not possible in your situation, you can enable the DNS challenge, which will disable the HTTP and TLS-ALPN challenges and use the DNS challenge exclusively.

Technically, only one challenge needs to be enabled for things to work, but using multiple is good for reliability in case a challenge is discontinued by the CA. This happened to the TLS-SNI challenge in early 2018&mdash;many popular ACME clients such as Traefik and Autocert broke, resulting in downtime for some sites, until new releases were made and patches deployed, because they used only one challenge; Caddy, however&mdash;this library's forerunner&mdash;was unaffected because it also used the HTTP challenge. If multiple challenges are enabled, they are chosen randomly to help prevent false reliance on a single challenge type. And if one fails, any remaining enabled challenges are tried before giving up.

### HTTP Challenge

Per the ACME spec, the HTTP challenge requires port 80, or at least packet forwarding from port 80. It works by serving a specific HTTP response that only the genuine server would have to a normal HTTP request at a special endpoint.

If you are running an HTTP server, solving this challenge is very easy: just wrap your handler in `HTTPChallengeHandler` _or_ call `SolveHTTPChallenge()` inside your own `ServeHTTP()` method.

For example, if you're using the standard library:

```go
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Lookit my cool website over HTTPS!")
})

http.ListenAndServe(":80", myACME.HTTPChallengeHandler(mux))
```

If wrapping your handler is not a good solution, try this inside your `ServeHTTP()` instead:

```go
magic := otomatik.NewDefault()
myACME := otomatik.NewACMEManager(magic, otomatik.DefaultACME)

func ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if myACME.HandleHTTPChallenge(w, r) {
		return // challenge handled; nothing else to do
	}
	...
}
```

If you are not running an HTTP server, you should disable the HTTP challenge _or_ run an HTTP server whose sole job it is to solve the HTTP challenge.

### TLS-ALPN Challenge

Per the ACME spec, the TLS-ALPN challenge requires port 443, or at least packet forwarding from port 443. It works by providing a special certificate using a standard TLS extension, Application Layer Protocol Negotiation (ALPN), having a special value. This is the most convenient challenge type because it usually requires no extra configuration and uses the standard TLS port which is where the certificates are used, also.

This challenge is easy to solve: just use the provided `tls.Config` when you make your TLS listener:

```go
// use this to configure a TLS listener
tlsConfig := magic.TLSConfig()
```

Or make two simple changes to an existing `tls.Config`:

```go
myTLSConfig.GetCertificate = magic.GetCertificate
myTLSConfig.NextProtos = append(myTLSConfig.NextProtos, tlsalpn01.ACMETLS1Protocol}
```

Then just make sure your TLS listener is listening on port 443:

```go
ln, err := tls.Listen("tcp", ":443", myTLSConfig)
```

### DNS Challenge

The DNS challenge is perhaps the most useful challenge because it allows you to obtain certificates without your server needing to be publicly accessible on the Internet, and it's the only challenge by which Let's Encrypt will issue wildcard certificates.

This challenge works by setting a special record in the domain's zone. To do this automatically, your DNS provider needs to offer an API by which changes can be made to domain names, and the changes need to take effect immediately for best results. otomatik supports [all of lego's DNS provider implementations](https://github.com/go-acme/lego/tree/master/providers/dns)! All of them clean up the temporary record after the challenge completes.

To enable it, just set the `DNSProvider` field on a `otomatik.Config` struct, or set the default `otomatik.DNSProvider` variable. For example, if my domains' DNS was served by DNSimple and I set my DNSimple API credentials in environment variables:

```go
import "github.com/go-acmeclient/lego/v3/providers/dns/dnsimple"

provider, err := dnsimple.NewDNSProvider()
if err != nil {
	return err
}

otomatik.DefaultACME.DNSProvider = provider
```

Now the DNS challenge will be used by default, and I can obtain certificates for wildcard domains. See the [pkg.go.dev documentation for the provider you're using](https://pkg.go.dev/github.com/go-acme/lego/providers/dns?tab=subdirectories) to learn how to configure it. Most can be configured by env variables or by passing in a config struct. If you pass a config struct instead of using env variables, you will probably need to set some other defaults (that's just how lego works, currently):

```go
PropagationTimeout: dns01.DefaultPollingInterval,
PollingInterval:    dns01.DefaultPollingInterval,
TTL:                dns01.DefaultTTL,
```

Enabling the DNS challenge disables the other challenges for that `otomatik.Config` instance.

## On-Demand TLS

Normally, certificates are obtained and renewed before a listener starts serving, and then those certificates are maintained throughout the lifetime of the program. In other words, the certificate names are static. But sometimes you don't know all the names ahead of time, or you don't want to manage all the certificates up front. This is where On-Demand TLS shines.

Originally invented for use in Caddy (which was the first program to use such technology), On-Demand TLS makes it possible and easy to serve certificates for arbitrary or specific names during the lifetime of the server. When a TLS handshake is received, otomatik will read the Server Name Indication (SNI) value and either load and present that certificate in the ServerHello, or if one does not exist, it will obtain it from a CA right then-and-there.

Of course, this has some obvious security implications. You don't want to DoS a CA or allow arbitrary clients to fill your storage with spammy TLS handshakes. That's why, when you enable On-Demand issuance, you should set limits or policy to allow getting certificates. otomatik has an implicit whitelist built-in which is sufficient for nearly everyone, but also has a more advanced way to control on-demand issuance.

The simplest way to enable on-demand issuance is to set the OnDemand field of a Config (or the default package-level value):

```go
otomatik.Default.OnDemand = new(otomatik.OnDemandConfig)
```

By setting this to a non-nil value, on-demand TLS is enabled for that config. For convenient security, otomatik's high-level abstraction functions such as `HTTPS()`, `TLS()`, `ManageSync()`, `ManageAsync()`, and `Listen()` (which all accept a list of domain names) will whitelist those names automatically so only certificates for those names can be obtained when using the Default config. Usually this is sufficient for most users.

However, if you require advanced control over which domains can be issued certificates on-demand (for example, if you do not know which domain names you are managing, or just need to defer their operations until later), you should implement your own DecisionFunc:

```go
// if the decision function returns an error, a certificate
// may not be obtained for that name at that time
otomatik.Default.OnDemand = &otomatik.OnDemandConfig{
	DecisionFunc: func(name string) error {
		if name != "example.com" {
			return fmt.Errorf("not allowed")
		}
		return nil
	},
}
```

The [pkg.go.dev](https://pkg.go.dev/github.com/caddyserver/otomatik?tab=doc#OnDemandConfig) describes how to use this in full detail, so please check it out!

## Storage

otomatik relies on storage to store certificates and other TLS assets (OCSP staple cache, coordinating locks, etc). Persistent storage is a requirement when using otomatik: ephemeral storage will likely lead to rate limiting on the CA-side as otomatik will always have to get new certificates.

By default, otomatik stores assets on the local file system in `$HOME/.local/share/otomatik` (and honors `$XDG_DATA_HOME` if set). otomatik will create the directory if it does not exist. If writes are denied, things will not be happy, so make sure otomatik can write to it!

The notion of a "cluster" or "fleet" of instances that may be serving the same site and sharing certificates, etc, is tied to storage. Simply, any instances that use the same storage facilities are considered part of the cluster. So if you deploy 100 instances of otomatik behind a load balancer, they are all part of the same cluster if they share the same storage configuration. Sharing storage could be mounting a shared folder, or implementing some other distributed storage system such as a database server or KV store.

The easiest way to change the storage being used is to set `otomatik.DefaultStorage` to a value that satisfies the [Storage interface](https://pkg.go.dev/github.com/caddyserver/otomatik?tab=doc#Storage). Keep in mind that a valid `Storage` must be able to implement some operations atomically in order to provide locking and synchronization.

If you write a Storage implementation, please add it to the [project wiki](https://github.com/caddyserver/otomatik/wiki/Storage-Implementations) so people can find it!

## Cache

All of the certificates in use are de-duplicated and cached in memory for optimal performance at handshake-time. This cache must be backed by persistent storage as described above.

Most applications will not need to interact with certificate caches directly. Usually, the closest you will come is to set the package-wide `otomatik.DefaultStorage` variable (before attempting to create any Configs). However, if your use case requires using different storage facilities for different Configs (that's highly unlikely and NOT recommended! Even Caddy doesn't get that crazy), you will need to call `otomatik.NewCache()` and pass in the storage you want to use, then get new `Config` structs with `otomatik.NewWithCache()` and pass in the cache.

Again, if you're needing to do this, you've probably over-complicated your application design.

## Credits and License

Otomatik is originally a folk of [CertMagic](https://github.com/caddyserver/otomatik), a project by [Matthew Holt](https://twitter.com/mholt6), who is the author; and other contributors, and now customised and maintained separately for internal use at [Chamaconekt Kenya](https://github.com/chamaconekt). Otomatik is licensed under [Apache 2.0](https://github.com/wondenge/otomatik/blob/master/LICENSE), an open source license.
