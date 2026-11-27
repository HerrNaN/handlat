package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
)

const metadataLimit = 32 << 10

var errClientMetadata = errors.New("invalid OAuth client metadata")

type clientMetadata struct {
	ID          string   `json:"client_id"`
	Name        string   `json:"client_name"`
	Redirects   []string `json:"redirect_uris"`
	AuthMethod  string   `json:"token_endpoint_auth_method"`
	AuthMethods []string `json:"token_endpoint_auth_methods_supported"`
}

func httpsClientID(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" ||
		u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") ||
		u.Path == "" || u.Path == "/" || (u.Port() != "" && u.Port() != "443") ||
		len(raw) > 2048 {
		return nil, false
	}

	return u, true
}

func readClient(ctx context.Context, httpClient *http.Client, id string) (clientMetadata, error) {
	var metadata clientMetadata

	u, ok := httpsClientID(id)
	if !ok {
		return metadata, errClientMetadata
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return metadata, errClientMetadata
	}

	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return metadata, errClientMetadata
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK || resp.ContentLength > metadataLimit {
		return metadata, errClientMetadata
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataLimit+1))
	if err != nil || len(body) > metadataLimit || json.Unmarshal(body, &metadata) != nil {
		return metadata, errClientMetadata
	}

	if metadata.ID != id || len(metadata.Redirects) == 0 ||
		(len(metadata.AuthMethods) != 0 && !slices.Contains(metadata.AuthMethods, "none")) ||
		(len(metadata.AuthMethods) == 0 && metadata.AuthMethod != "" && metadata.AuthMethod != "none") {
		return metadata, errClientMetadata
	}

	metadata.Name = strings.TrimSpace(metadata.Name)
	if metadata.Name == "" || len(metadata.Name) > 200 ||
		strings.ContainsFunc(
			metadata.Name,
			func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) },
		) {
		metadata.Name = u.Hostname()
	}

	return metadata, nil
}

func redirectMatches(registered, requested string) bool {
	a, errA := url.Parse(registered)

	b, errB := url.Parse(requested)
	if errA != nil || errB != nil || len(requested) > 2048 ||
		b.User != nil || b.Hostname() == "" || strings.Contains(requested, "#") ||
		b.Opaque != "" {
		return false
	}

	if b.Scheme == "https" {
		return registered == requested
	}

	// RFC 8252 allows a native client to choose its loopback port at runtime.
	ip, err := netip.ParseAddr(b.Hostname())
	if err != nil || !ip.IsLoopback() || ip.Zone() != "" {
		return false
	}

	if b.Scheme != "http" || b.Port() == "" || a.User != nil ||
		strings.Contains(registered, "#") {
		return false
	}

	a.Host, b.Host = a.Hostname(), b.Hostname()

	return a.String() == b.String()
}

func metadataHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext:            dialPublic,
			DisableKeepAlives:      true,
			MaxResponseHeaderBytes: 16 << 10,
		},
	}
}

var restrictedNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

var publicIPv6 = netip.MustParsePrefix("2000::/3")

func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()

	return ip.IsGlobalUnicast() && !ip.IsPrivate() &&
		(ip.Is4() || publicIPv6.Contains(ip)) &&
		!slices.ContainsFunc(
			restrictedNetworks,
			func(prefix netip.Prefix) bool { return prefix.Contains(ip) },
		)
}

func dialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, errClientMetadata
	}

	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 ||
		slices.ContainsFunc(ips, func(ip netip.Addr) bool { return !isPublic(ip) }) {
		return nil, errClientMetadata
	}

	// Dial the checked addresses directly to avoid a second DNS lookup.
	dialer := net.Dialer{Timeout: 2 * time.Second}
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}

	return nil, errClientMetadata
}
