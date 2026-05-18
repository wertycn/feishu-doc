package core

import (
	"fmt"
	"io"
	"os"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

type Client struct {
	larkClient      *lark.Client
	UserAccessToken string
	Logger          io.Writer
	Domain          string
}

func NewClient(appID, appSecret, domain string) *Client {
	var opts []lark.ClientOptionFunc
	if domain != "" && strings.Contains(domain, "larksuite.com") {
		opts = append(opts, lark.WithOpenBaseUrl("https://open.larksuite.com"))
	}
	return &Client{
		larkClient: lark.NewClient(appID, appSecret, opts...),
		Logger:     os.Stderr,
		Domain:     domain,
	}
}

func (c *Client) DocURL(docID string) string {
	if c.Domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/docx/%s", c.Domain, docID)
}

func (c *Client) WikiNodeURL(nodeToken string) string {
	if c.Domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/wiki/%s", c.Domain, nodeToken)
}

func (c *Client) RequestOpts() []larkcore.RequestOptionFunc {
	if c.UserAccessToken != "" {
		return []larkcore.RequestOptionFunc{larkcore.WithUserAccessToken(c.UserAccessToken)}
	}
	return nil
}

func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func DerefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func DerefBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
