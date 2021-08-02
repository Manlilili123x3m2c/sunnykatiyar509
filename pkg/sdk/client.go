package sdk

import (
	"path"
	"path/filepath"

	"cocogo/pkg/common"
	"cocogo/pkg/config"
)

type ClientAuth interface {
	Sign() string
}

type WrapperClient struct {
	Http     *common.Client
	Auth     ClientAuth
	BaseHost string
}

func (c *WrapperClient) LoadAuth() error {
	keyPath := config.Conf.AccessKeyFile
	if !path.IsAbs(config.Conf.AccessKeyFile) {
		keyPath = filepath.Join(config.Conf.RootPath, keyPath)
	}
	ak := AccessKey{Value: config.Conf.AccessKey, Path: keyPath}
	err := ak.Load()
	if err != nil {
		return err
	}
	c.Auth = ak
	return nil
}

func (c *WrapperClient) CheckAuth() error {
	var user User
	err := c.Http.Get("UserProfileUrl", &user)
	if err != nil {
		return err
	}
	return nil
}

func (c *WrapperClient) Get(url string, res interface{}, needAuth bool) error {
	if needAuth {
		c.Http.SetAuth(c.Auth.Sign())
	} else {
		c.Http.SetAuth("")
	}

	return c.Http.Get(c.BaseHost+url, res)
}

func (c *WrapperClient) Post(url string, data interface{}, res interface{}, needAuth bool) error {
	if needAuth {
		c.Http.SetAuth(c.Auth.Sign())
	} else {
		c.Http.SetAuth("")
	}
	return c.Http.Post(url, data, res)
}

func (c *WrapperClient) Delete(url string, res interface{}, needAuth bool) error {
	if needAuth {
		c.Http.SetAuth(c.Auth.Sign())
	} else {
		c.Http.SetAuth("")
	}
	return c.Http.Delete(url, res)
}

func (c *WrapperClient) Put(url string, data interface{}, res interface{}, needAuth bool) error {
	if needAuth {
		c.Http.SetAuth(c.Auth.Sign())
	} else {
		c.Http.SetAuth("")
	}
	return c.Http.Put(url, data, res)
}

func (c *WrapperClient) Patch(url string, data interface{}, res interface{}, needAuth bool) error {
	if needAuth {
		c.Http.SetAuth(c.Auth.Sign())
	} else {
		c.Http.SetAuth("")
	}
	return c.Http.Patch(url, data, res)
}
