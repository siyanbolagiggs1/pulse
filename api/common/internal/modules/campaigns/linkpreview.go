package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pulse/api/internal/database"
	"golang.org/x/net/html"
)

const (
	linkPreviewCachePrefix  = "link_preview:"
	linkPreviewCacheTTL     = 24 * time.Hour
	linkPreviewMaxBody      = 200 * 1024 // 200KB — plenty for a <head> block
	linkPreviewTimeout      = 5 * time.Second
	linkPreviewMaxRedirects = 3
)

var ErrLinkPreviewBlocked = errors.New("target URL is not fetchable")

type LinkPreview struct {
	Title       string `json:"title"`
	Image       string `json:"image"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	URL         string `json:"url"`
}

// safeIP rejects loopback, private, link-local, and unspecified addresses so
// the link-preview fetcher can't be used to reach internal services — the
// target URL comes from a business-authored campaign field, not a trusted
// source.
func safeIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast())
}

// safeDialContext resolves the host itself and dials the resolved IP
// directly (rather than letting net/http resolve-then-connect separately),
// so there's no DNS-rebinding gap between validating the address and
// connecting to it. This runs for the initial request AND every redirect
// hop, since it's wired into the Transport rather than duplicated in
// CheckRedirect.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	var dialIP net.IP
	for _, ip := range ips {
		if safeIP(ip) {
			dialIP = ip
			break
		}
	}
	if dialIP == nil {
		return nil, fmt.Errorf("no fetchable address for %s", host)
	}
	d := net.Dialer{Timeout: linkPreviewTimeout}
	return d.DialContext(ctx, network, net.JoinHostPort(dialIP.String(), port))
}

func newLinkPreviewClient() *http.Client {
	return &http.Client{
		Timeout:   linkPreviewTimeout,
		Transport: &http.Transport{DialContext: safeDialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= linkPreviewMaxRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

func fetchLinkPreview(ctx context.Context, rawURL string) (*LinkPreview, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, ErrLinkPreviewBlocked
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PulseLinkPreview/1.0)")

	resp, err := newLinkPreviewClient().Do(req)
	if err != nil {
		return nil, ErrLinkPreviewBlocked
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrLinkPreviewBlocked
	}

	preview := parseOpenGraph(io.LimitReader(resp.Body, linkPreviewMaxBody))
	preview.Domain = parsed.Hostname()
	preview.URL = rawURL
	return preview, nil
}

// parseOpenGraph scans for og:title/og:image/og:description meta tags,
// falling back to <title> when no og:title is present.
func parseOpenGraph(r io.Reader) *LinkPreview {
	preview := &LinkPreview{}
	tokenizer := html.NewTokenizer(r)
	inTitle := false

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return preview

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "meta":
				var property, name, content string
				for _, a := range token.Attr {
					switch a.Key {
					case "property":
						property = a.Val
					case "name":
						name = a.Val
					case "content":
						content = a.Val
					}
				}
				key := property
				if key == "" {
					key = name
				}
				switch key {
				case "og:title":
					preview.Title = content
				case "og:image":
					preview.Image = content
				case "og:description", "description":
					if preview.Description == "" {
						preview.Description = content
					}
				}
			case "title":
				if preview.Title == "" {
					inTitle = true
				}
			}

		case html.TextToken:
			if inTitle {
				preview.Title = strings.TrimSpace(string(tokenizer.Text()))
				inTitle = false
			}

		case html.EndTagToken:
			if tokenizer.Token().Data == "title" {
				inTitle = false
			}
		}
	}
}

// getLinkPreview looks up the campaign's targetUrl and returns its cached or
// freshly-fetched Open Graph preview. A fetch failure degrades to a
// domain-only preview rather than an error — the campaign card should never
// fail to render just because its target site is slow or blocks scraping.
func getLinkPreview(ctx context.Context, campaignID string) (*LinkPreview, error) {
	campaign, err := getCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	cacheKey := linkPreviewCachePrefix + campaign.TargetURL
	if database.Redis != nil {
		if cached, err := database.Redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var hit LinkPreview
			if json.Unmarshal(cached, &hit) == nil {
				return &hit, nil
			}
		}
	}

	preview, err := fetchLinkPreview(ctx, campaign.TargetURL)
	if err != nil {
		preview = &LinkPreview{URL: campaign.TargetURL}
		if parsed, perr := url.Parse(campaign.TargetURL); perr == nil {
			preview.Domain = parsed.Hostname()
		}
	}

	if database.Redis != nil {
		if b, merr := json.Marshal(preview); merr == nil {
			database.Redis.Set(ctx, cacheKey, b, linkPreviewCacheTTL)
		}
	}

	return preview, nil
}
