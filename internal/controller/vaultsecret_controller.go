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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vaultv1alpha1 "github.com/apexdriver/vault-configurator/api/v1alpha1"
	vault "github.com/apexdriver/vault-configurator/internal/vault"
)

// VaultSecretReconciler reconciles a VaultSecret object
type VaultSecretReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	VaultClient vault.VaultClient
}

func NewVaultSecretReconciler(mgr ctrl.Manager, vc vault.VaultClient) *VaultSecretReconciler {
	return &VaultSecretReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		VaultClient: vc,
	}
}

// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=vaultsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=vaultsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=vaultsecrets/finalizers,verbs=update

func (r *VaultSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var vs vaultv1alpha1.VaultSecret
	if err := r.Get(ctx, req.NamespacedName, &vs); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	client := r.VaultClient

	mounts, err := client.ListMounts()
	if err != nil {
		log.Error(err, "failed to list Vault mounts")
		return ctrl.Result{}, err
	}

	for _, mount := range vs.Spec.Mounts {
		mountPath := mount.Mount
		if !strings.HasSuffix(mountPath, "/") {
			mountPath += "/"
		}

		mountConfig, exists := mounts[mountPath]
		if !exists {
			log.Error(nil, "mount path not found in Vault", "mountPath", mountPath)
			return ctrl.Result{}, nil
		}

		isV2 := false
		if mountConfig.Type == "kv" {
			if version, ok := mountConfig.Options["version"]; ok && version == "2" {
				isV2 = true
			}
		}

		for _, s := range mount.Secrets {
			if isV2 {
				log.Info("Detected KV v2 secret engine", "mount", mount.Mount, "secret", s.Path)

				exists, err = client.ReadSecretV2(mount.Mount, s.Path)
				if err != nil {
					log.Error(err, "failed to check if v2 secret exists", "mount", mount.Mount, "secret", s.Path)
					return ctrl.Result{}, err
				}
				if exists {
					log.Info("Secret already exists, skipping write", "mount", mount.Mount, "secret", s.Path)
				}

				err = client.WriteSecretV2(mount.Mount, s.Path, s.Data.Raw)

			} else {
				log.Info("Detected KV v1 secret engine", "mount", mount.Mount, "secret", s.Path)

				exists, err = client.ReadSecretV1(mount.Mount, s.Path)
				if err != nil {
					log.Error(err, "failed too check if v1 secret exists", "mount", mount.Mount, "secret", s.Path)
					return ctrl.Result{}, err
				}
				if exists {
					log.Info("Secret already exists, skipping write", "mount", mount.Mount, "secret", s.Path)
					continue
				}

				err = client.WriteSecretV1(mount.Mount, s.Path, s.Data.Raw)
			}

			if err != nil {
				log.Error(err, "failed to write secret", "mount", mount.Mount, "secret", s.Path)
			}
			log.Info("Secret written successfully", "mount", mount.Mount, "secret", s.Path)
		}
	}

	vs.Status.Created = true
	if err := r.Status().Update(ctx, &vs); err != nil {
		log.Error(err, "failed to update VaultSecret status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultv1alpha1.VaultSecret{}).
		Named("vaultsecret").
		Complete(r)
}
