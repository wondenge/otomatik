package otomatik

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"reflect"
	"testing"
)

func TestCertificateResource_NamesKey(t *testing.T) {
	type fields struct {
		SANs           []string
		CertificatePEM []byte
		PrivateKeyPEM  []byte
		IssuerData     interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CertificateResource{
				SANs:           tt.fields.SANs,
				CertificatePEM: tt.fields.CertificatePEM,
				PrivateKeyPEM:  tt.fields.PrivateKeyPEM,
				IssuerData:     tt.fields.IssuerData,
			}
			if got := cr.NamesKey(); got != tt.want {
				t.Errorf("NamesKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPS(t *testing.T) {
	type args struct {
		domainNames []string
		mux         http.Handler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := HTTPS(tt.args.domainNames, tt.args.mux); (err != nil) != tt.wantErr {
				t.Errorf("HTTPS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListen(t *testing.T) {
	type args struct {
		domainNames []string
	}
	tests := []struct {
		name    string
		args    args
		want    net.Listener
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Listen(tt.args.domainNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("Listen() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Listen() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManageAsync(t *testing.T) {
	type args struct {
		ctx         context.Context
		domainNames []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ManageAsync(tt.args.ctx, tt.args.domainNames); (err != nil) != tt.wantErr {
				t.Errorf("ManageAsync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManageSync(t *testing.T) {
	type args struct {
		domainNames []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ManageSync(tt.args.domainNames); (err != nil) != tt.wantErr {
				t.Errorf("ManageSync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnDemandConfig_whitelistContains(t *testing.T) {
	type fields struct {
		DecisionFunc  func(name string) error
		hostWhitelist []string
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OnDemandConfig{
				DecisionFunc:  tt.fields.DecisionFunc,
				hostWhitelist: tt.fields.hostWhitelist,
			}
			if got := o.whitelistContains(tt.args.name); got != tt.want {
				t.Errorf("whitelistContains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTLS(t *testing.T) {
	type args struct {
		domainNames []string
	}
	tests := []struct {
		name    string
		args    args
		want    *tls.Config
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TLS(tt.args.domainNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("TLS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TLS() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hostOnly(t *testing.T) {
	type args struct {
		hostport string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostOnly(tt.args.hostport); got != tt.want {
				t.Errorf("hostOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_httpRedirectHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_isInternal(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternal(tt.args.addr); got != tt.want {
				t.Errorf("isInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isLoopback(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLoopback(tt.args.addr); got != tt.want {
				t.Errorf("isLoopback() = %v, want %v", got, tt.want)
			}
		})
	}
}