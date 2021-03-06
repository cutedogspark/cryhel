package cryhel

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
)

// any is purely semantic
type any interface{}

// Pointer is purely semantic
type pointer interface{}

func isPointer(value any) bool {
	if reflect.ValueOf(value).Kind() != reflect.Ptr {
		return false
	}
	return true
}

// encrypt general func
func (c *Crypto) encrypt(msg string) ([]byte, error) {
	plaintext := ZeroPadding([]byte(msg))

	if len(plaintext)%aes.BlockSize != 0 {
		return nil, errors.New("plaintext is not a multiple of the block size")
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, errors.New(err.Error())
	}

	mode := cipher.NewCBCEncrypter(c.block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// decrypt general func
func (c *Crypto) decrypt(ciphertext []byte) ([]byte, error) {
	blockMode := cipher.NewCBCDecrypter(c.block, c.bkey[:c.block.BlockSize()])
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	planeText := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(planeText, ciphertext)

	planeText = ZeroUnPadding(planeText[aes.BlockSize:])
	return planeText, nil
}

func ZeroPadding(in []byte) []byte {
	padding := aes.BlockSize - (len(in) % aes.BlockSize)
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(in, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData,
		func(r rune) bool {
			return r == rune(0)
		})
}

// Crypto struct
type Crypto struct {
	block cipher.Block
	bkey  []byte

	Encrypt *EncryptService
	Decrypt *DecryptService
}

func NewCrypto(secretkey string) (c *Crypto, err error) {
	if secretkey == "" {
		err = errors.New("secret key empty")
		return
	}

	c = &Crypto{}
	c.bkey = []byte(secretkey)
	c.block, err = aes.NewCipher(c.bkey)
	c.Encrypt = NewEncryptService(c)
	c.Decrypt = NewDecryptService(c)
	if err != nil {
		return
	}
	return
}

//-----------------------------------------------------------------------------
// EncryptService
type EncryptService struct {
	s *Crypto
}

func NewEncryptService(s *Crypto) *EncryptService {
	rs := &EncryptService{s: s}
	return rs
}

// method  "Crypto.Encrypt.Msg"
type EncryptMsgCall struct {
	s *Crypto
	m string
}

func (r *EncryptService) Msg(msg string) *EncryptMsgCall {
	c := &EncryptMsgCall{s: r.s, m: msg}
	return c
}

func (r *EncryptMsgCall) Do() (string, error) {
	if ciphertext, err := r.s.encrypt(r.m); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(ciphertext), nil
	}
}

// method  "Crypto.Encrypt.QueryEscapeMsg"
type EncryptQueryEscapeMsgCall struct {
	s *Crypto
	m string
}

func (r *EncryptService) QueryEscapeMsg(msg string) *EncryptQueryEscapeMsgCall {
	c := &EncryptQueryEscapeMsgCall{s: r.s, m: msg}
	return c
}

func (r *EncryptQueryEscapeMsgCall) Do() (string, error) {
	if ciphertext, err := r.s.encrypt(r.m); err != nil {
		return "", err
	} else {
		base64EncodeString := url.QueryEscape(base64.StdEncoding.EncodeToString(ciphertext))

		return base64EncodeString, nil
	}
}

//-----------------------------------------------------------------------------
// DecryptService
type DecryptService struct {
	s *Crypto
}

func NewDecryptService(s *Crypto) *DecryptService {
	rs := &DecryptService{s: s}
	return rs
}

// method  "Crypto.Decrypt.Msg"
type DecryptMsgCall struct {
	s *Crypto
	m string
}

func (r *DecryptService) Msg(msg string) *DecryptMsgCall {
	c := &DecryptMsgCall{s: r.s, m: msg}
	return c
}

func (r *DecryptMsgCall) Do() (string, error) {
	ciphertext, _ := base64.StdEncoding.DecodeString(r.m) // decrypt base64
	buf, err := r.s.decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (r *DecryptMsgCall) Out(out pointer) error {
	if !isPointer(out) {
		return errors.New(fmt.Sprintf("Value '%s' is not a pointer", out))
	}

	ciphertext, _ := base64.StdEncoding.DecodeString(r.m) // decrypt base64
	buf, err := r.s.decrypt(ciphertext)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, out); err != nil {
		return err
	} else {
		return nil
	}
}

// method  "Crypto.Decrypt.QueryEscapeMsg"
type DecryptQueryEscapeMsgCall struct {
	s *Crypto
	m string
}

func (r *DecryptService) QueryEscapeMsg(msg string) *DecryptQueryEscapeMsgCall {
	c := &DecryptQueryEscapeMsgCall{s: r.s, m: msg}
	return c
}

func (r *DecryptQueryEscapeMsgCall) Do() (string, error) {
	base64Encode, _ := url.QueryUnescape(r.m)                      // url unescape
	ciphertext, _ := base64.StdEncoding.DecodeString(base64Encode) // decrypt base64
	buf, err := r.s.decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (r *DecryptQueryEscapeMsgCall) Out(out pointer) error {
	if !isPointer(out) {
		return errors.New(fmt.Sprintf("Value '%s' is not a pointer", out))
	}

	base64Encode, _ := url.QueryUnescape(r.m)                      // url unescape
	ciphertext, _ := base64.StdEncoding.DecodeString(base64Encode) // decrypt base64
	buf, err := r.s.decrypt(ciphertext)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, out); err != nil {
		return err
	} else {
		return nil
	}
}

//-----------------------------------------------------------------------------
