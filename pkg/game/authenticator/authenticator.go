package authenticator

import (
	"fmt"
	"net/url"
)

type AuthenticatorType string

const (
	MICROSOFT AuthenticatorType = "msa"
	MOJANG    AuthenticatorType = "mojang"
	CUSTOM    AuthenticatorType = "custom"
	UNKNOWN   AuthenticatorType = "unknown"
)

type AuthenticatorResponse struct {
	UserUUID    string            `json:"user_uuid"`
	Token       string            `json:"token"`
	UserName    string            `json:"username"`
	OtherTokens map[string]string `json:"other_tokens"`
}

type Authenticator interface {
	GetType() AuthenticatorType
	AuthenticateWithCredentials(username string, password string) (*AuthenticatorResponse, error)
	AuthenticateWithCode(code string) (*AuthenticatorResponse, error)
}

func FindAuthenticatorFromURI(uri string) (Authenticator, *url.URL, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, nil, err
	}

	switch parsed.Scheme {
	case "microsoft":
		return NewMicrosoftAuthenticator(), parsed, nil
	case "custom":
		return NewCustomAuthenticator(CustomAuthenticatorConfig{
			BaseURL:       parsed.Host,
			LoginEndpoint: parsed.Path,
		}), parsed, nil
	default:
		return nil, nil, fmt.Errorf("invalid scheme: %s", parsed.Scheme)
	}
}
