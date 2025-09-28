package profile

import (
	"strconv"

	"limeal.fr/launchygo/game/authenticator"
)

type Memory struct {
	Xmx int `json:"xmx"` // The maximum memory to use in GB
	Xms int `json:"xms"` // The minimum memory to use in GB
}

func (m *Memory) ToArgs() []string {
	return []string{
		"-Xmx" + strconv.Itoa(m.Xmx) + "G",
		"-Xms" + strconv.Itoa(m.Xms) + "G",
	}
}

type GameProfile struct {
	Username        string  `json:"username"`
	UUID            *string `json:"uuid"`
	UserType        string  `json:"userType"`
	Token           string  `json:"token"`
	Memory          Memory  `json:"memory"`
	IsAuthenticated bool    `json:"isAuthenticated"`
}

func NewGameProfile() *GameProfile {
	return &GameProfile{
		Username:        "steve",
		UUID:            nil,
		UserType:        string(authenticator.UNKNOWN),
		Memory:          Memory{Xmx: 4, Xms: 2},
		IsAuthenticated: false,
	}
}

func (g *GameProfile) SetMemory(xmx int, xms int) {
	g.Memory.Xmx = xmx
	g.Memory.Xms = xms
}

func (g *GameProfile) SetUser(username string) {
	g.Username = username
	g.UserType = string(authenticator.MOJANG)
}

func (g *GameProfile) AuthenticateWithCredentials(authenticator authenticator.Authenticator, username string, password string) error {
	response, err := authenticator.AuthenticateWithCredentials(username, password)
	if err != nil {
		return err
	}

	g.UUID = &response.UserUUID
	g.Username = response.UserName
	g.UserType = string(authenticator.GetType())
	g.Token = response.Token
	g.IsAuthenticated = true
	return nil
}

func (g *GameProfile) AuthenticateWithCode(authenticator authenticator.Authenticator, code string) error {
	response, err := authenticator.AuthenticateWithCode(code)
	if err != nil {
		return err
	}

	g.UUID = &response.UserUUID
	g.Username = response.UserName
	g.UserType = string(authenticator.GetType())
	g.Token = response.Token
	g.IsAuthenticated = true
	return nil
}
