package authenticator

import (
	"fmt"
	"net/http"
	"strconv"

	"limeal.fr/launchygo/pkg/utils"
)

////////////////////////////////////////////////////////////
// Constants & Types
////////////////////////////////////////////////////////////

type CustomAuthenticationRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail"`
	Password        string `json:"password"`
}

type CustomAuthenticationResponse struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	AccessToken string `json:"accessToken"`
}

type CustomAuthenticatorConfig struct {
	BaseURL       string
	LoginEndpoint string
}

////////////////////////////////////////////////////////////
// Authenticator
////////////////////////////////////////////////////////////

type CustomAuthenticator struct {
	config CustomAuthenticatorConfig
}

func NewCustomAuthenticator(config CustomAuthenticatorConfig) *CustomAuthenticator {
	return &CustomAuthenticator{config: config}
}

func (a *CustomAuthenticator) GetType() AuthenticatorType {
	return CUSTOM
}

func (a *CustomAuthenticator) AuthenticateWithCode(code string) (*AuthenticatorResponse, error) {
	return nil, nil
}

func (a *CustomAuthenticator) AuthenticateWithCredentials(email string, password string) (*AuthenticatorResponse, error) {
	fmt.Println("Authenticating with Chronos")

	bodyResponse := CustomAuthenticationResponse{}
	requestOptions := utils.NewRequestOptions[CustomAuthenticationResponse]("application/json", &bodyResponse)
	requestOptions.AddHeader("Accept", "application/json")
	requestOptions.SetBody(CustomAuthenticationRequest{
		UsernameOrEmail: email,
		Password:        password,
	})

	_, err := utils.DoRequest(http.MethodPost, a.config.BaseURL+a.config.LoginEndpoint, requestOptions)
	if err != nil {
		return nil, err
	}

	return &AuthenticatorResponse{
		UserUUID: strconv.Itoa(bodyResponse.UserID),
		Token:    bodyResponse.AccessToken,
		UserName: bodyResponse.Username,
	}, nil
}
