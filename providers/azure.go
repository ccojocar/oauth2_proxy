package providers

import (
	"errors"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/bitly/oauth2_proxy/api"
	"log"
	"net/http"
	"net/url"
)

type AzureProvider struct {
	*ProviderData
	Tenant         string
	GroupValidator func(*SessionState) bool
}

func NewAzureProvider(p *ProviderData) *AzureProvider {
	p.ProviderName = "Azure"

	if p.ProfileURL == nil || p.ProfileURL.String() == "" {
		p.ProfileURL = &url.URL{
			Scheme:   "https",
			Host:     "graph.windows.net",
			Path:     "/me",
			RawQuery: "api-version=1.6",
		}
	}
	if p.ProtectedResource == nil || p.ProtectedResource.String() == "" {
		p.ProtectedResource = &url.URL{
			Scheme: "https",
			Host:   "graph.windows.net",
		}
	}
	if p.Scope == "" {
		p.Scope = "openid"
	}

	return &AzureProvider{
		ProviderData: p,
		// Set the default group validator to allow any group. It will be overwritten by a
		// specific validator when group restriction is configured.
		GroupValidator: func(s *SessionState) bool {
			return true
		},
	}
}

func (p *AzureProvider) Configure(tenant string) {
	p.Tenant = tenant
	if tenant == "" {
		p.Tenant = "common"
	}

	if p.LoginURL == nil || p.LoginURL.String() == "" {
		p.LoginURL = &url.URL{
			Scheme: "https",
			Host:   "login.microsoftonline.com",
			Path:   "/" + p.Tenant + "/oauth2/authorize"}
	}
	if p.RedeemURL == nil || p.RedeemURL.String() == "" {
		p.RedeemURL = &url.URL{
			Scheme: "https",
			Host:   "login.microsoftonline.com",
			Path:   "/" + p.Tenant + "/oauth2/token",
		}
	}
}

func getAzureHeader(access_token string) http.Header {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", access_token))
	return header
}

func getEmailFromJSON(json *simplejson.Json) (string, error) {
	var email string
	var err error

	email, err = json.Get("mail").String()

	if err != nil || email == "" {
		otherMails, otherMailsErr := json.Get("otherMails").Array()
		if len(otherMails) > 0 {
			email = otherMails[0].(string)
		}
		err = otherMailsErr
	}

	return email, err
}

func (p *AzureProvider) GetEmailAddress(s *SessionState) (string, error) {
	var email string
	var err error

	if s.AccessToken == "" {
		return "", errors.New("missing access token")
	}
	req, err := http.NewRequest("GET", p.ProfileURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header = getAzureHeader(s.AccessToken)

	json, err := api.Request(req)

	if err != nil {
		return "", err
	}

	email, err = getEmailFromJSON(json)

	if err == nil && email != "" {
		return email, err
	}

	email, err = json.Get("userPrincipalName").String()

	if err != nil {
		log.Printf("failed making request %s", err)
		return "", err
	}

	if email == "" {
		log.Printf("failed to get email address")
		return "", err
	}

	return email, err
}

// SetGroupRestriction configures the AzureProvider to restrict access to the specified groups.
func (p *AzureProvider) SetGroupRestriction(groups []string) {

}

// ValidateGroup validates that the user provided in the SessionState is a member of the configured groups.
func (p *AzureProvider) ValidateGroup(s *SessionState) bool {
	return p.GroupValidator(s)
}
