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
	"encoding/json"
	"fmt"
	"path"

	"github.com/go-logr/logr"
	vaultAPI "github.com/hashicorp/vault/api"
)

// Interface with interacting with vault server
type VaultClient interface {
	PutPolicy(name string, policy string) error
	ReadSecretV1(mountPath, path string) (bool, error)
	ReadSecretV2(mountPath, path string) (bool, error)
	WriteSecretV1(mountPath, path string, data json.RawMessage) error
	WriteSecretV2(mountPath, path string, data json.RawMessage) error
	Mount(path string, input *vaultAPI.MountInput) error
	ListMounts() (map[string]*vaultAPI.MountOutput, error)
	ListAuth() (map[string]*vaultAPI.MountOutput, error)
	EnableAuthWithOptions(path string, input *vaultAPI.MountInput) error
	WriteAuthConfig(mountPath string, config map[string]interface{}) error
	WriteAuthRole(mountPath string, roleName string, roledata map[string]interface{}) error
	MountSecretEngine(mountPath, engineType string, options map[string]string) error
}

// RealVaultClient implements VaultClient using the actual Vault API client
type RealVaultClient struct {
	client *vaultAPI.Client
	auth   Authenticator
	log    logr.Logger
}

// NewClient creates and returns an authenticated Vault client using the provided role and logger.
func NewClient(role string, logger logr.Logger) (VaultClient, error) {
	// Initialize Vault client with default config (no address yet)
	client, err := vaultAPI.NewClient(vaultAPI.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vault client: %w", err)
	}

	// Create an authenticator with the specified role
	auth := NewServiceAccountTokenAuthenticator(role)

	// Perform login (sets address and token)
	if err := auth.Login(client, logger); err != nil {
		return nil, fmt.Errorf("vault login failed: %w", err)
	}

	return &RealVaultClient{
		client: client,
		auth:   auth,
		log:    logger,
	}, nil
}

func (v *RealVaultClient) PutPolicy(name string, policy string) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	return v.client.Sys().PutPolicy(name, policy)
}

func (v *RealVaultClient) ReadSecretV1(mountPath, secretPath string) (bool, error) {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return false, err
	}

	fullPath := path.Join(mountPath, secretPath)
	secret, err := v.client.Logical().Read(fullPath)
	if err != nil {
		v.log.Error(err, "failed to read v1 secret", "mount", mountPath, "secret", secretPath)
		return false, err
	}
	return secret != nil, nil
}

func (v *RealVaultClient) ReadSecretV2(mountPath, secretPath string) (bool, error) {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return false, err
	}

	fullPath := path.Join(mountPath, "data", secretPath)
	secret, err := v.client.Logical().Read(fullPath)
	if err != nil {
		v.log.Error(err, "failed to read v2 secret", "mount", mountPath, "secret", secretPath)
		return false, err
	}
	return secret != nil, nil
}

func (v *RealVaultClient) WriteSecretV1(mountPath, secretPath string, raw json.RawMessage) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		v.log.Error(err, "failed to unmarshal v1 data", "path", secretPath)
		return err
	}

	fullPath := path.Join(mountPath, secretPath)
	_, err := v.client.Logical().Write(fullPath, data)
	if err != nil {
		v.log.Error(err, "failed to write v1 secret", "mount", mountPath, "path", secretPath)
	}
	return err
}

func (v *RealVaultClient) WriteSecretV2(mountPath, secretPath string, raw json.RawMessage) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		v.log.Error(err, "failed to unmarshal v2 data", "path", secretPath)
		return err
	}
	fullPath := path.Join(mountPath, "data", secretPath)
	_, err := v.client.Logical().Write(fullPath, map[string]interface{}{"data": data})
	if err != nil {
		v.log.Error(err, "failed to write v2 secret", "mount", mountPath, "path", secretPath)
	}
	return err
}

func (v *RealVaultClient) Mount(mountPath string, input *vaultAPI.MountInput) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	return v.client.Sys().Mount(mountPath, input)
}

func (v *RealVaultClient) ListMounts() (map[string]*vaultAPI.MountOutput, error) {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return nil, err
	}

	return v.client.Sys().ListMounts()
}

func (v *RealVaultClient) ListAuth() (map[string]*vaultAPI.MountOutput, error) {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return nil, err
	}

	authMethods, err := v.client.Sys().ListAuth()
	if err != nil {
		v.log.Error(err, "failed to list auth methods")
		return nil, err
	}
	return authMethods, nil
}

func (v *RealVaultClient) EnableAuthWithOptions(path string, input *vaultAPI.MountInput) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	err := v.client.Sys().EnableAuthWithOptions(path, input)
	if err != nil {
		v.log.Error(err, "failed to enable auth method", "path", path)
	}
	return err
}

func (v *RealVaultClient) WriteAuthConfig(mountPath string, config map[string]interface{}) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	configPath := fmt.Sprintf("auth/%s/config", mountPath)
	v.log.Info("Writing Kubernetes auth config", "path", configPath)

	_, err := v.client.Logical().Write(configPath, config)
	if err != nil {
		v.log.Error(err, "failed to write Kubernetes auth config", "path", configPath)
	}
	return err
}

func (v *RealVaultClient) WriteAuthRole(mountPath, roleName string, roleData map[string]interface{}) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	rolePath := fmt.Sprintf("auth/%s/role/%s", mountPath, roleName)
	v.log.Info("Writing Kubernetes auth role", "path", rolePath, "role", roleName)

	_, err := v.client.Logical().Write(rolePath, roleData)
	if err != nil {
		v.log.Error(err, "failed to write Kubernetes auth role", "path", rolePath)
	}
	return err
}

func (v *RealVaultClient) MountSecretEngine(mountPath, engineType string, options map[string]string) error {
	if err := v.auth.Login(v.client, v.log); err != nil {
		return err
	}

	input := &vaultAPI.MountInput{
		Type:    engineType,
		Options: options,
	}
	return v.client.Sys().Mount(mountPath, input)
}
