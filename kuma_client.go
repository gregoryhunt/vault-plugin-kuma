package kuma

import "fmt"

type kumaUser struct {
	Username string
	Password string
}

// A kuma database client used as an example, in a real world this
// would be an external library provided by a database provider.
type KumaClient struct {
	Username string
	Password string
	URL      string
	users    map[string]kumaUser
}

func NewKumaClient(url, username, password string) (KumaClient, error) {
	return KumaClient{URL: url, Username: username, Password: password, users: make(map[string]kumaUser)}, nil
}

func (c *KumaClient) CreateUser(username, password string) kumaUser {
	user := kumaUser{Username: username, Password: password}
	c.users[username] = user
	return user
}

func (c *KumaClient) UpdateUser(username, password string) error {
	if val, ok := c.users[username]; ok {
		val.Password = password
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
