package ufile

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"bytes"
)

type Context struct {
	PrivateKey string
	PublicKey  string
	Bucket     string
}

func (c *Context) getSignature(method string, key string) (string) {
	string2sign := fmt.Sprint(method, "\n", "", "\n", "", "\n", "", "\n", "/", c.Bucket, "/", key)
	/* hmac-sha1 */
	mac := hmac.New(sha1.New, []byte(c.PrivateKey))
	mac.Write([]byte(string2sign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature
}

func (c *Context) Put(id string, data []byte) ([]byte, error) {
	signature := c.getSignature("PUT", id)
	client := &http.Client{}
	req, err := http.NewRequest("PUT", "http://"+c.Bucket+".ufile.ucloud.cn/"+id, bytes.NewReader(data))
	req.Header.Add("Authorization", "UCloud "+c.PublicKey+":"+signature)
	req.Header.Add("Content-Length", string(len(data)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s\n", string(body))
	return []byte(body), nil
}

func (c *Context) Get(id string) ([]byte, error) {
	signature := c.getSignature("GET", id)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://"+c.Bucket+".ufile.ucloud.cn/"+id, nil)
	req.Header.Add("Authorization", "UCloud "+c.PublicKey+":"+signature)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return []byte(body), nil
}
