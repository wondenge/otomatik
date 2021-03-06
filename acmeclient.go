package otomatik

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	weakrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v3/acme"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/registration"
)

func init() {
	weakrand.Seed(time.Now().UnixNano())
}

// acmeClient is a wrapper over lego's acme.Client with some custom state attached.
// It is used to obtain, renew, and revoke certificates with ACME.
// Use ACMEManager.newACMEClient() or ACMEManager.newACMEClientWithRetry() to get a valid one for real use.
type acmeClient struct {
	caURL      string
	mgr        *ACMEManager
	acmeClient *lego.Client
	challenges []challenge.Type
}

// newACMEClientWithRetry is the same as newACMEClient, but with automatic retry capabilities.
// Sometimes network connections or HTTP requests fail intermittently, even when requesting the directory endpoint.
// For example, so we can avoid that by just retrying once.
// Failures here are rare and sporadic, usually, so a simple retry is an easy fix.
func (manager *ACMEManager) newACMEClientWithRetry(useTestCA bool) (*acmeClient, error) {
	var client *acmeClient
	var err error
	const maxTries = 2
	for i := 0; i < maxTries; i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		client, err = manager.newACMEClient(useTestCA, false) // TODO: move logic that requires interactivity to way before this part of the process...
		if err == nil {
			break
		}
		if acmeErr, ok := err.(acme.ProblemDetails); ok {
			if acmeErr.HTTPStatus == http.StatusTooManyRequests {
				return nil, fmt.Errorf("too many requests making new ACME client: %+v - aborting", acmeErr)
			}
		}
		log.Printf("[ERROR] Making new ACME client: %v (attempt %d/%d)", err, i+1, maxTries)
	}
	return client, err
}

// newACMEClient creates the underlying ACME library client type.
// If useTestCA is true, am.TestCA will be used if it is set;
// otherwise, the primary CA will still be used.
func (manager *ACMEManager) newACMEClient(useTestCA, interactive bool) (*acmeClient, error) {
	acmeClientsMu.Lock()
	defer acmeClientsMu.Unlock()

	// ensure defaults are filled in
	certObtainTimeout := manager.CertObtainTimeout
	if certObtainTimeout == 0 {
		certObtainTimeout = DefaultACME.CertObtainTimeout
	}
	var caURL string
	if useTestCA {
		caURL = manager.TestCA
		// Only use the default test CA if the CA is also the default CA;
		// no point in testing against Let's Encrypt's staging server if we are not using their production server too.
		if caURL == "" && manager.CA == DefaultACME.CA {
			caURL = DefaultACME.TestCA
		}
	}
	if caURL == "" {
		caURL = manager.CA
	}
	if caURL == "" {
		caURL = DefaultACME.CA
	}

	// ensure endpoint is secure (assume HTTPS if scheme is missing)
	if !strings.Contains(caURL, "://") {
		caURL = "https://" + caURL
	}
	u, err := url.Parse(caURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !isLoopback(u.Host) && !isInternal(u.Host) {
		return nil, fmt.Errorf("%s: insecure CA URL (HTTPS required)", caURL)
	}

	// look up or create the user account
	leUser, err := manager.getUser(caURL, manager.Email)
	if err != nil {
		return nil, err
	}

	// if a lego client with this configuration already exists, reuse it
	clientKey := caURL + leUser.Email
	client, ok := acmeClients[clientKey]
	if !ok {
		// the client facilitates our communication with the CA server
		legoCfg := lego.NewConfig(leUser)
		legoCfg.CADirURL = caURL
		legoCfg.UserAgent = buildUAString()
		legoCfg.HTTPClient.Timeout = HTTPTimeout
		legoCfg.Certificate = lego.CertificateConfig{
			Timeout: manager.CertObtainTimeout,
		}
		if manager.TrustedRoots != nil {
			if ht, ok := legoCfg.HTTPClient.Transport.(*http.Transport); ok {
				if ht.TLSClientConfig == nil {
					ht.TLSClientConfig = new(tls.Config)
					ht.ForceAttemptHTTP2 = true
				}
				ht.TLSClientConfig.RootCAs = manager.TrustedRoots
			}
		}
		client, err = lego.NewClient(legoCfg)
		if err != nil {
			return nil, err
		}
		acmeClients[clientKey] = client
	}

	// if not registered, the user must register an account with the CA and agree to terms
	if leUser.Registration == nil {
		if interactive { // can't prompt a user who isn't there
			termsURL := client.GetToSURL()
			if !manager.Agreed && termsURL != "" {
				manager.Agreed = manager.askUserAgreement(client.GetToSURL())
			}
			if !manager.Agreed && termsURL != "" {
				return nil, fmt.Errorf("user must agree to CA terms")
			}
		}

		var reg *registration.Resource
		if manager.ExternalAccount != nil {
			reg, err = client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
				TermsOfServiceAgreed: manager.Agreed,
				Kid:                  manager.ExternalAccount.KeyID,
				HmacEncoded:          base64.StdEncoding.EncodeToString(manager.ExternalAccount.HMAC),
			})
		} else {
			reg, err = client.Registration.Register(registration.RegisterOptions{
				TermsOfServiceAgreed: manager.Agreed,
			})
		}
		if err != nil {
			return nil, err
		}
		leUser.Registration = reg

		// persist the user to storage
		err = manager.saveUser(caURL, leUser)
		if err != nil {
			return nil, fmt.Errorf("could not save user: %v", err)
		}
	}

	c := &acmeClient{
		caURL:      caURL,
		mgr:        manager,
		acmeClient: client,
	}

	return c, nil
}

// initialChallenges returns the initial set of challenges to try using c.config as a basis.
func (client *acmeClient) initialChallenges() []challenge.Type {
	// if configured, use DNS challenge exclusively
	if client.mgr.DNSProvider != nil {
		return []challenge.Type{challenge.DNS01}
	}

	// otherwise, use HTTP and TLS-ALPN challenges if enabled
	var chal []challenge.Type
	if !client.mgr.DisableHTTPChallenge {
		chal = append(chal, challenge.HTTP01)
	}
	if !client.mgr.DisableTLSALPNChallenge {
		chal = append(chal, challenge.TLSALPN01)
	}
	return chal
}

// nextChallenge chooses a challenge randomly from the given list of available challenges
// and configures client.acmeClient to use that challenge according to client.config.
// It pops the chosen challenge from the list and returns that challenge along with the new list without that challenge.
// If len(available) == 0, this is a no-op.
// Don't even get me started on how dumb it is we need to do this here instead of the upstream lego library doing it for us.
// Lego used to randomize the challenge order, thus allowing another one to be used if the first one failed.
// https://github.com/go-acme/lego/issues/842
// (It also has an awkward API for adjusting the available challenges.)
// At time of writing, lego doesn't try anything other than the TLS-ALPN challenge, even if the HTTP challenge is also enabled.
// So we take matters into our own hands and enable only one challenge at a time in the underlying client, randomly selected by us.
func (client *acmeClient) nextChallenge(available []challenge.Type) (challenge.Type, []challenge.Type) {
	if len(available) == 0 {
		return "", available
	}

	// make sure we choose a challenge randomly, which lego used to do but
	// the critical feature was surreptitiously removed in ~2018 in a commit
	// too large to review, oh well - choose one, then remove it from the
	// list of available challenges so it doesn't get retried
	randIdx := weakrand.Intn(len(available))
	randomChallenge := available[randIdx]
	available = append(available[:randIdx], available[randIdx+1:]...)

	// clean the slate, since we reuse clients
	client.acmeClient.Challenge.Remove(challenge.HTTP01)
	client.acmeClient.Challenge.Remove(challenge.TLSALPN01)
	client.acmeClient.Challenge.Remove(challenge.DNS01)

	switch randomChallenge {
	case challenge.HTTP01:
		useHTTPPort := HTTPChallengePort
		if HTTPPort > 0 && HTTPPort != HTTPChallengePort {
			useHTTPPort = HTTPPort
		}
		if client.mgr.AltHTTPPort > 0 {
			useHTTPPort = client.mgr.AltHTTPPort
		}

		client.acmeClient.Challenge.SetHTTP01Provider(distributedSolver{
			acmeManager: client.mgr,
			providerServer: &httpSolver{
				acmeManager: client.mgr,
				address:     net.JoinHostPort(client.mgr.ListenHost, strconv.Itoa(useHTTPPort)),
			},
			caURL: client.caURL,
		})

	case challenge.TLSALPN01:
		useTLSALPNPort := TLSALPNChallengePort
		if HTTPSPort > 0 && HTTPSPort != TLSALPNChallengePort {
			useTLSALPNPort = HTTPSPort
		}
		if client.mgr.AltTLSALPNPort > 0 {
			useTLSALPNPort = client.mgr.AltTLSALPNPort
		}

		client.acmeClient.Challenge.SetTLSALPN01Provider(distributedSolver{
			acmeManager: client.mgr,
			providerServer: &tlsALPNSolver{
				config:  client.mgr.config,
				address: net.JoinHostPort(client.mgr.ListenHost, strconv.Itoa(useTLSALPNPort)),
			},
			caURL: client.caURL,
		})

	case challenge.DNS01:
		if client.mgr.DNSChallengeOption != nil {
			client.acmeClient.Challenge.SetDNS01Provider(client.mgr.DNSProvider, client.mgr.DNSChallengeOption)
		} else {
			client.acmeClient.Challenge.SetDNS01Provider(client.mgr.DNSProvider)
		}
	}

	return randomChallenge, available
}

func (client *acmeClient) throttle(ctx context.Context, names []string) error {
	// throttling is scoped to CA + account email
	rateLimiterKey := client.caURL + "," + client.mgr.Email
	rateLimitersMu.Lock()
	rl, ok := rateLimiters[rateLimiterKey]
	if !ok {
		rl = NewRateLimiter(RateLimitEvents, RateLimitEventsWindow)
		rateLimiters[rateLimiterKey] = rl
		// TODO: stop rate limiter when it is garbage-collected...
	}
	rateLimitersMu.Unlock()
	log.Printf("[INFO]%v Waiting on rate limiter...", names)
	err := rl.Wait(ctx)
	if err != nil {
		return err
	}
	log.Printf("[INFO]%v Done waiting", names)
	return nil
}

func (client *acmeClient) usingTestCA() bool {
	return client.mgr.TestCA != "" && client.caURL == client.mgr.TestCA
}

func (client *acmeClient) revoke(_ context.Context, certRes certificate.Resource) error {
	return client.acmeClient.Certificate.Revoke(certRes.Certificate)
}

func buildUAString() string {
	ua := "otomatik"
	if UserAgent != "" {
		ua += " " + UserAgent
	}
	return ua
}

// These internal rate limits are designed to prevent accidentally firehosing a CA's ACME endpoints.
// They are not intended to replace or replicate the CA's actual rate limits.
//
// Let's Encrypt's rate limits can be found here:
// https://letsencrypt.org/docs/rate-limits/
//
// Currently (as of December 2019), Let's Encrypt's most relevant rate limit for large
// deployments is 300 new orders per account per 3 hours (on average, or best case, that's
// about 1 every 36 seconds, or 2 every 72 seconds, etc.); but it's not reasonable to try to
// assume that our internal state is the same as the CA's (due to process restarts, config changes,
// failed validations, etc.) and ultimately, only the CA's actual rate limiter is the authority.
//
// Thus, our own rate limiters do not attempt to enforce external rate limits.
// Doing so causes problems when the domains are not in our control (i.e. serving customer sites)
// and/or lots of domains fail validation: they clog our internal rate limiter and nearly starve
// out (or at least slow down) the other domains that need certificates.
//
// Failed transactions are already retried with exponential backoff,
// so adding in rate limiting can slow things down even more.
//
// Instead, the point of our internal rate limiter is to avoid hammering the CA's
// endpoint when there are thousands or even millions of certificates under management.
// Our goal is to allow small bursts in a relatively short timeframe so as to not block
// any one domain for too long, without unleashing thousands of requests to the CA at once.
var (
	rateLimiters   = make(map[string]*RingBufferRateLimiter)
	rateLimitersMu sync.RWMutex

	// RateLimitEvents is how many new events can be allowed in RateLimitEventsWindow.
	RateLimitEvents = 10

	// RateLimitEventsWindow is the size of the sliding window that throttles events.
	RateLimitEventsWindow = 1 * time.Minute
)

// Some default values passed down to the underlying lego client.
var (
	UserAgent   string
	HTTPTimeout = 30 * time.Second
)

// We keep a global cache of ACME clients so that they can be reused.
// Since the number of CAs, accounts, and key types should be fairly limited under best practices,
// this map will hardly ever have more than a few entries at the most.
// The associated lock protects access to the map but also ensures that only one ACME client is created at a time.
// TODO: consider using storage for a distributed lock
// TODO: consider evicting clients after some time
var (
	acmeClients   = make(map[string]*lego.Client)
	acmeClientsMu sync.Mutex
)
