package authenticator

import (
	"fmt"
	"log"
	"net/http"

	"limeal.fr/launchygo/utils"
)

////////////////////////////////////////////////////////////
// Constants & Types
////////////////////////////////////////////////////////////

const (
	MicrosoftAuthorizeURL = "https://login.live.com/oauth20_authorize.srf"
	MicrosoftTokenURL     = "https://login.live.com/oauth20_token.srf"

	XBoxLiveAuthenticateURL = "https://user.auth.xboxlive.com/user/authenticate"
	XBoxAuthMethod          = "RPS"
	XBoxSiteName            = "user.auth.xboxlive.com"
	XBoxRelyingParty        = "http://auth.xboxlive.com"
	XBoxTokenType           = "JWT"

	XSTSAuthorizeURL = "https://xsts.auth.xboxlive.com/xsts/authorize"
	XSTSSandboxID    = "RETAIL"
	XSTSRelyingParty = "rp://api.minecraftservices.com/"
	XSTSTokenType    = "JWT"

	MojangAuthorizeURL = "https://api.minecraftservices.com/authentication/login_with_xbox"

	MicrosoftScope     = "service::user.auth.xboxlive.com::MBI_SSL"
	MicrosoftGrantType = "authorization_code"

	MojangClientID    = "00000000402B5328"
	MojangRedirectURI = "https://login.live.com/oauth20_desktop.srf"
	MojangUserInfoURL = "https://api.minecraftservices.com/minecraft/profile"
)

type MicrosoftTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// Xbox Authentication Request

type XboxAuthenticationRequestProperties struct {
	AuthMethod string `json:"AuthMethod"`
	SiteName   string `json:"SiteName"`
	RpsTicket  string `json:"RpsTicket"`
}

type XboxAuthenticationRequest struct {
	Properties   XboxAuthenticationRequestProperties `json:"Properties"`
	RelyingParty string                              `json:"RelyingParty"`
	TokenType    string                              `json:"TokenType"`
}

type XboxAuthenticationResponse struct {
	IssueInstant  string `json:"IssueInstant"`
	NotAfter      string `json:"NotAfter"`
	Token         string `json:"Token"`
	DisplayClaims struct {
		Xui []struct {
			Uhs string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

// XSTS Authentication Request

type XSTSAuthenticationRequestProperties struct {
	SandboxId  string   `json:"SandboxId"`
	UserTokens []string `json:"UserTokens"`
}

type XSTSAuthenticationRequest struct {
	Properties XSTSAuthenticationRequestProperties `json:"Properties"`

	RelyingParty string `json:"RelyingParty"`
	TokenType    string `json:"TokenType"`
}

// Minecraft Login Request

type MinecraftLoginRequest struct {
	IdentityToken string `json:"IdentityToken"`
}

// Minecraft Login Response

type MinecraftLoginResponse struct {
	ID          string `json:"username"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// User Info Request

type UserInfoResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"properties"`
}

////////////////////////////////////////////////////////////
// Microsoft Authenticator
////////////////////////////////////////////////////////////

type MicrosoftAuthenticator struct {
}

func NewMicrosoftAuthenticator() *MicrosoftAuthenticator {
	return &MicrosoftAuthenticator{}
}

func (a *MicrosoftAuthenticator) GetType() AuthenticatorType {
	return MICROSOFT
}

func (a *MicrosoftAuthenticator) GetAuthorizationURL() string {
	return fmt.Sprintf("%s?prompt=select_account&client_id=%s&response_type=code&scope=%s&redirect_uri=%s&lw=1&fl=dob&easi2&xsup=1&nopa=2", MicrosoftAuthorizeURL, MojangClientID, MicrosoftScope, MojangRedirectURI)
}

func (a *MicrosoftAuthenticator) AuthenticateWithCredentials(username string, password string) (*AuthenticatorResponse, error) {
	return nil, nil
}

func (a *MicrosoftAuthenticator) AuthenticateWithCode(code string) (*AuthenticatorResponse, error) {
	tokens := MicrosoftTokenResponse{}

	// 1. Get Microsoft Token
	fmt.Println("[*] Getting Microsoft Token")
	optionsMicrosoft := utils.NewRequestOptions[MicrosoftTokenResponse]("application/x-www-form-urlencoded", &tokens)
	optionsMicrosoft.SetBody(map[string]string{
		"code":         code,
		"grant_type":   MicrosoftGrantType,
		"scope":        MicrosoftScope,
		"client_id":    MojangClientID,
		"redirect_uri": MojangRedirectURI,
	})

	_, err := utils.DoRequest(http.MethodPost, MicrosoftTokenURL, optionsMicrosoft)
	if err != nil {
		log.Fatal("failed to get microsoft token: ", err)
		return nil, err
	}

	// 2. Get Xbox Token

	fmt.Println("[*] Getting Xbox Token")
	xboxAuthResponse := XboxAuthenticationResponse{}
	optionsXbox := utils.NewRequestOptions[XboxAuthenticationResponse]("application/json", &xboxAuthResponse)
	optionsXbox.AddHeader("Accept", "application/json")
	optionsXbox.SetBody(XboxAuthenticationRequest{
		Properties: XboxAuthenticationRequestProperties{
			AuthMethod: XBoxAuthMethod,
			SiteName:   XBoxSiteName,
			RpsTicket:  tokens.AccessToken,
		},
		RelyingParty: XBoxRelyingParty,
		TokenType:    XBoxTokenType,
	})

	_, err = utils.DoRequest(http.MethodPost, XBoxLiveAuthenticateURL, optionsXbox)
	if err != nil {
		log.Fatal("failed to get xbox token: ", err)
		return nil, err
	}

	// 3. Get XSTS Token

	fmt.Println("[*] Getting XSTS Token")
	xstsAuthResponse := XboxAuthenticationResponse{}
	optionsXsts := utils.NewRequestOptions[XboxAuthenticationResponse]("application/json", &xstsAuthResponse)
	optionsXsts.AddHeader("Accept", "application/json")
	optionsXsts.SetBody(XSTSAuthenticationRequest{
		Properties: XSTSAuthenticationRequestProperties{
			SandboxId:  XSTSSandboxID,
			UserTokens: []string{xboxAuthResponse.Token},
		},
		RelyingParty: XSTSRelyingParty,
		TokenType:    XSTSTokenType,
	})

	_, err = utils.DoRequest(http.MethodPost, XSTSAuthorizeURL, optionsXsts)
	if err != nil {
		log.Fatal("failed to get xsts token: ", err)
		return nil, err
	}

	// 4. Get Minecraft Token

	fmt.Println("[*] Getting Minecraft Token")
	minecraftLoginResponse := MinecraftLoginResponse{}
	optionsMinecraft := utils.NewRequestOptions[MinecraftLoginResponse]("application/json", &minecraftLoginResponse)
	optionsMinecraft.SetBody(MinecraftLoginRequest{
		IdentityToken: fmt.Sprintf("XBL3.0 x=%s;%s", xboxAuthResponse.DisplayClaims.Xui[0].Uhs, xstsAuthResponse.Token),
	})

	_, err = utils.DoRequest(http.MethodPost, MojangAuthorizeURL, optionsMinecraft)
	if err != nil {
		log.Fatal("failed to get minecraft token: ", err)
		return nil, err
	}

	// 5. Get User Info

	fmt.Println("[*] Getting User Info")
	userInfo := UserInfoResponse{}
	optionsUserInfo := utils.NewRequestOptions[UserInfoResponse]("application/json", &userInfo)
	optionsUserInfo.AddHeader("Authorization", fmt.Sprintf("Bearer %s", minecraftLoginResponse.AccessToken))
	_, err = utils.DoRequest(http.MethodGet, MojangUserInfoURL, optionsUserInfo)
	if err != nil {
		log.Fatal("failed to get user info: ", err)
		return nil, err
	}

	return &AuthenticatorResponse{
		UserUUID: userInfo.ID,
		Token:    minecraftLoginResponse.AccessToken,
		UserName: userInfo.Name,
		OtherTokens: map[string]string{
			"microsoft": tokens.AccessToken,
			"xbox":      xboxAuthResponse.Token,
			"xsts":      xstsAuthResponse.Token,
		},
	}, nil
}
