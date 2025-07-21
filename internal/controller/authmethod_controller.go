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

package controller

import (
	"context"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vaultv1alpha1 "github.com/apexdriver/vault-configurator/api/v1alpha1"
	vault "github.com/apexdriver/vault-configurator/internal/vault"
	vaultAPI "github.com/hashicorp/vault/api"
)

// AuthMethodReconciler reconciles a AuthMethod object
type AuthMethodReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	VaultClient vault.VaultClient
}

func NewAuthMethodReconciler(mgr ctrl.Manager, vc vault.VaultClient) *AuthMethodReconciler {
	return &AuthMethodReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		VaultClient: vc,
	}
}

// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=authmethods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=authmethods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=authmethods/finalizers,verbs=update

func (r *AuthMethodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var authMethod vaultv1alpha1.AuthMethod
	if err := r.Get(ctx, req.NamespacedName, &authMethod); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	client := r.VaultClient

	authMethods, err := client.ListAuth()
	if err != nil {
		log.Error(err, "failed to list auth methods")
		return ctrl.Result{}, err
	}

	for _, authMethod := range authMethod.Spec.AuthMethods {
		log.Info("Processing AuthMethod", "path", authMethod.Path, "type", authMethod.Type)

		// Enable auth method if not present
		if _, exists := authMethods[authMethod.Path+"/"]; !exists {
			log.Info("Enabling auth method", "path", authMethod.Path)
			if err := client.EnableAuthWithOptions(authMethod.Path, &vaultAPI.MountInput{
				Type: authMethod.Type,
			}); err != nil {
				log.Error(err, "failed to enable auth method", "path", authMethod.Path)
				return ctrl.Result{}, err
			}
		}

		// Delegate to Kubernetes auth handler
		if authMethod.Type == "kubernetes" {
			if err := r.configureKubernetesAuth(ctx, client, authMethod.Path, authMethod.Roles); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// TODO: Add support for other auth method types as needed

	return ctrl.Result{}, nil
}

func (r *AuthMethodReconciler) configureKubernetesAuth(ctx context.Context, client vault.VaultClient, path string, roles []vaultv1alpha1.AuthMethodRole) error {
	log := ctrl.LoggerFrom(ctx)
	config := map[string]interface{}{
		"kubernetes_host": "https://kubernetes.default.svc",
	}

	// Read CA cert
	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Error(err, "failed to read Kubernetes CA cert")
		return err
	}
	config["kubernetes_ca_cert"] = string(caCert)

	// Read SA token for token reviewer
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Error(err, "failed to read ServiceAccount token for token reviewer")
		return err
	}
	config["token_reviewer_jwt"] = string(token)

	// Write Kubernetes auth config
	if err := client.WriteAuthConfig(path, config); err != nil {
		return err
	}

	// Configure each role
	for _, role := range roles {
		roleData := map[string]interface{}{
			"bound_service_account_names":      role.ServiceAccount,
			"bound_service_account_namespaces": role.Namespaces,
			"policies":                         role.Policies,
			"token_ttl":                        "1h",
			"token_max_ttl":                    "2h",
		}

		if err := client.WriteAuthRole(path, role.Name, roleData); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthMethodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultv1alpha1.AuthMethod{}).
		Named("authmethod").
		Complete(r)
}
