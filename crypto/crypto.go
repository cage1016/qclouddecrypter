package crypto

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/qeek-dev/cryhel"
)

type Credentials struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Provider     string `json:"provider"`
	Error        string `json:"error"`
	Scope        string `json:"scope"`
}

type SignedAuth struct {
	Token string `json:"token"`
}

type CryptoServicer interface {
	// decrypt Credentials
	DecryptCredentials(result string) (c Credentials, err error)
	// Encrypt Struct
	EncryptStruct(in interface{}) (string, error)
	// Encrypt Message
	Encrypt(msg string) (encString string, err error)
	// Encrypt Message
	Decrypt(encString string) (sourceString string, err error)
	// DecryptToStruct Message
	DecryptToStruct(encString string, out interface{}) error
	// Encrypt Signed Url
	EncryptSignedUrl(in interface{}) (string, error)
	// Decrypt Singed Url
	DecryptSignedUrl(encString string) (toke, sourceString string, err error)
}

type cryptoService struct {
	c *cryhel.Crypto
}

func (r *cryptoService) DecryptCredentials(result string) (c Credentials, err error) {
	s, err := url.QueryUnescape(result)
	if err != nil {
		err = errors.New(fmt.Sprintf("QueryUnescape fail: %v", err))
		return
	}

	t := url.QueryEscape(s)
	if t == result {
		err = r.c.Decrypt.Msg(s).Out(&c)
		if err != nil {
			err = errors.New(fmt.Sprintf("Decrypt fail: %v", err))
		}
		return
	} else {
		r.c.Decrypt.Msg(result).Encoding(base64.StdEncoding).Out(&c)
		return
	}
}

func (r *cryptoService) DecryptResultFromCloudConnectorServer(result string) (c Credentials, err error) {
	base64Encode, _ := url.QueryUnescape(result)
	err = r.c.Decrypt.Msg(base64Encode).Out(&c)
	return
}

func (r *cryptoService) DecryptRefreshResultFromCloudConnectorServer(result string) (c Credentials, err error) {
	err = r.c.Decrypt.Msg(result).Encoding(base64.StdEncoding).Out(&c)
	return
}

func (r *cryptoService) Encrypt(msg string) (encString string, err error) {
	return r.c.Encrypt.Msg(msg).Encoding(base64.RawURLEncoding).Do()
}

func (r *cryptoService) EncryptStruct(in interface{}) (string, error) {
	bytes, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return r.c.Encrypt.Msg(string(bytes)).Encoding(base64.RawURLEncoding).Do()
}

func (r *cryptoService) Decrypt(encString string) (sourceString string, err error) {
	return r.c.Decrypt.Msg(encString).Encoding(base64.RawURLEncoding).Do()
}

func (r *cryptoService) DecryptToStruct(encString string, out interface{}) error {
	return r.c.Decrypt.Msg(encString).Encoding(base64.RawURLEncoding).Out(&out)
}

func (r *cryptoService) EncryptSignedUrl(in interface{}) (string, error) {
	bytes, _ := json.Marshal(in)

	var k SignedAuth
	json.Unmarshal(bytes, &k)
	if k.Token == "" {
		return "", errors.New("Signed Auth need `Token` property and non empty")
	}

	return r.c.Encrypt.Msg(string(bytes)).Encoding(base64.RawURLEncoding).Do()
}

func (r *cryptoService) DecryptSignedUrl(encString string) (toke, sourceString string, err error) {
	sourceString, err = r.c.Decrypt.Msg(encString).Encoding(base64.RawURLEncoding).Do()
	if err != nil {
		return "", "", err
	}
	var k SignedAuth
	err = json.Unmarshal([]byte(sourceString), &k)
	if err != nil {
		return "", "", err
	}
	return k.Token, sourceString, nil
}

func NewCryptoService(secretkey string) (*cryptoService, error) {
	r, err := cryhel.NewCrypto(secretkey)
	if err != nil {
		return nil, err
	}
	return &cryptoService{
		c: r,
	}, nil
}
