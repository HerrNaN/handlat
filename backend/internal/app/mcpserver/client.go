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

const (
	metadataLimit      = 32 << 10
	metadataTimeout    = 5 * time.Second
	maxMetadataFetches = 8
)

var (
	errClientMetadata      = errors.New("invalid OAuth client metadata")
	errClientIDMismatch    = errors.New("OAuth client metadata ID mismatch")
	errMetadataUnavailable = errors.New("OAuth client metadata unavailable")
	errMetadataBusy        = errors.New("OAuth client metadata fetch limit reached")
)

type clientMetadata struct {
	ID            string   `json:"client_id"`
	Name          string   `json:"client_name"`
	Redirects     []string `json:"redirect_uris"`
	AuthMethod    string   `json:"token_endpoint_auth_method"`
	AuthMethods   []string `json:"token_endpoint_auth_methods_supported"`
	GrantTypes    []string `json:"grant_types"`
	ResponseTypes []string `json:"response_types"`
}

func (c clientMetadata) supportsCode() bool {
	return slices.Contains(c.GrantTypes, "authorization_code") &&
		slices.Contains(c.ResponseTypes, "code")
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

func (h *Handler) lookupClient(ctx context.Context, id string) (clientMetadata, error) {
	u, ok := httpsClientID(id)
	if !ok {
		return clientMetadata{}, errClientMetadata
	}

	result := h.metadataLookups.DoChan(id, func() (any, error) {
		select {
		case h.metadataSlots <- struct{}{}:
		default:
			return clientMetadata{}, errMetadataBusy
		}

		defer func() { <-h.metadataSlots }()

		// A browser closing the request must not cancel a shared lookup or revocation.
		lookupCtx := context.WithoutCancel(ctx)

		metadata, err := readClient(lookupCtx, h.metadata, id, u)
		if errors.Is(err, errClientIDMismatch) {
			revokeCtx, cancel := context.WithTimeout(lookupCtx, metadataTimeout)
			defer cancel()

			if err := h.invalidateClientRefresh(revokeCtx, id); err != nil {
				return metadata, err
			}
		}

		return metadata, err
	})
	select {
	case <-ctx.Done():
		return clientMetadata{}, errMetadataUnavailable
	case lookup := <-result:
		metadata, _ := lookup.Val.(clientMetadata)
		return metadata, lookup.Err
	}
}

func readClient(
	ctx context.Context,
	httpClient *http.Client,
	id string,
	u *url.URL,
) (clientMetadata, error) {
	metadata := clientMetadata{
		GrantTypes: []string{"authorization_code"}, ResponseTypes: []string{"code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return metadata, errClientMetadata
	}

	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		if errors.Is(err, errClientMetadata) {
			return metadata, errClientMetadata
		}

		return metadata, errMetadataUnavailable
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusRequestTimeout ||
		resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode >= http.StatusInternalServerError {
		return metadata, errMetadataUnavailable
	}

	if resp.StatusCode != http.StatusOK || resp.ContentLength > metadataLimit {
		return metadata, errClientMetadata
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataLimit+1))
	if err != nil {
		return metadata, errMetadataUnavailable
	}

	if len(body) > metadataLimit || json.Unmarshal(body, &metadata) != nil {
		return metadata, errClientMetadata
	}

	if metadata.ID != "" && metadata.ID != id {
		return metadata, errClientIDMismatch
	}

	if metadata.ID == "" || len(metadata.Redirects) == 0 ||
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
		Timeout: metadataTimeout,
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
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 ||
		slices.ContainsFunc(ips, func(ip netip.Addr) bool { return !isPublic(ip) }) {
		return nil, errClientMetadata
	}

	// Dial the checked addresses directly to avoid a second DNS lookup.
	dialer := net.Dialer{Timeout: 2 * time.Second}

	var dialErr error

	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}

		dialErr = err
	}

	return nil, dialErr
}
