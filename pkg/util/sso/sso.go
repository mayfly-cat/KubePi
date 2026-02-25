package sso

import (
	"errors"
	"net/http"
)

type Sso struct {
	Enable           bool   `json:"enable"`
	Protocol         string `json:"protocol"`
	InterfaceAddress string `json:"interfaceAddress"`
	ClientId         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
}

func NewSsoClient(protocol, interfaceAddress, clientId, clientSecret string, enable bool) *Sso {
	return &Sso{
		Protocol:         protocol,
		InterfaceAddress: interfaceAddress,
		ClientId:         clientId,
		ClientSecret:     clientSecret,
		Enable:           enable,
	}
}

func (s *Sso) TestConnect(protocol, interfaceAddress string) error {
	//根据协议设置请求URL
	switch protocol {
	case "openid":
		interfaceAddress += "/.well-known/openid-configuration"
	case "saml2":
		interfaceAddress += "/sso/saml2/metadata"
	default:
		return errors.New("不支持的协议类型: " + protocol)
	}
	req, err := http.NewRequest("GET", interfaceAddress, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 判断是否增加成功
	if resp.StatusCode != 200 {
		return errors.New("请求SSO接口失败,当前状态码为: " + resp.Status)
	}
	return nil
}
