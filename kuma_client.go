package kuma

import "fmt"

type kumaUser struct {
	Username string //TODO might need a better name as the represents Users and Services
	Token    string
}

// A kuma control plane client
type KumaClient struct {
	Username string
	Token    string
	URL      string
	users    map[string]kumaUser //TODO User and Services both leverage tokens
}

func NewKumaClient(url, username, token string) (KumaClient, error) {
	return KumaClient{URL: url, Username: username, Token: token, users: make(map[string]kumaUser)}, nil
}

func (c *KumaClient) CreateUser(username, token string) kumaUser {
	user := kumaUser{Username: username, Token: token}
	c.users[username] = user
	return user
}

func (c *KumaClient) UpdateUser(username, token string) error {
	if val, ok := c.users[username]; ok {
		val.Token = token
		c.users[username] = val
		return nil
	}

	return fmt.Errorf("user does not exist")
}

func (c *KumaClient) DeleteUser(username string) error {
	if _, ok := c.users[username]; ok {
		delete(c.users, username)
		return nil
	}

	return fmt.Errorf("user does not exist")
}
