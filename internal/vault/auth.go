/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vault

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	vaultAPI "github.com/hashicorp/vault/api"
)

type Authenticator interface {
	Login(*vaultAPI.Client, logr.Logger) error
}

const tokenExpiryThreshold = 5 * time.Minute

type serviceAccountTokenAuthenticator struct {
	expiryTime time.Time
	role       string
}

func NewServiceAccountTokenAuthenticator(role string) Authenticator {
	return &serviceAccountTokenAuthenticator{
		role: role,
	}
}

func (auth *serviceAccountTokenAuthenticator) Login(client *vaultAPI.Client, logger logr.Logger) error {
	if !isAboutToExpire(auth.expiryTime) {
		return nil
	}

	vaultAddress := os.Getenv("VAULT_ADDR")
	if vaultAddress == "" {
		return fmt.Errorf("VAULT_ADDR is not set")
	}

	if err := client.SetAddress(vaultAddress); err != nil {
		return err
	}

	bearerToken, err := bearerToken()
	if err != nil {
		return err
	}

	loginData := map[string]interface{}{
		"jwt":  bearerToken,
		"role": auth.role,
	}

	loginPath := loginPath()
	if err != nil {
		return err
	}

	secret, err := client.Logical().Write(loginPath, loginData)
	if err != nil {
		logger.Error(err, "failed to login to vault")
		return err
	}

	auth.expiryTime, err = expiryTime(secret, logger)
	if err != nil {
		return err
	}

	client.SetToken(secret.Auth.ClientToken)

	logger.Info("Successfully logged into vault.", "expires", auth.expiryTime)
	return nil
}

func isAboutToExpire(expiryTime time.Time) bool {
	return time.Now().Add(tokenExpiryThreshold).After(expiryTime)
}

func expiryTime(secret *vaultAPI.Secret, logger logr.Logger) (time.Time, error) {
	expireTime := time.Time{}
	if secret.Auth.LeaseDuration <= 0 {
		err := fmt.Errorf("secret has invalid lease duration: %d", secret.Auth.LeaseDuration)
		logger.Error(err, "failed to get vault token expiry date")
		return expireTime, err
	}
	return time.Now().Add(time.Duration(secret.Auth.LeaseDuration) * time.Second), nil
}

func loginPath() string {
	loginPath := os.Getenv("VAULT_LOGIN_PATH")
	if loginPath == "" {
		loginPath = "auth/kubernetes/login"
	}
	return loginPath
}

func bearerToken() (string, error) {
	serviceAccountTokenPath, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}

	if string(serviceAccountTokenPath) == "" {
		return "", errors.New("bearer token is empty")
	}

	return string(serviceAccountTokenPath), nil
}
