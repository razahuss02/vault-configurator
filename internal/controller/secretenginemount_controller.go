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

// SecretEngineMountReconciler reconciles a SecretEngineMount object
type SecretEngineMountReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	VaultClient vault.VaultClient
}

func NewSecretEngineMountReconciler(mgr ctrl.Manager, vc vault.VaultClient) *SecretEngineMountReconciler {
	return &SecretEngineMountReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		VaultClient: vc,
	}
}

// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=secretenginemounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=secretenginemounts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=secretenginemounts/finalizers,verbs=update

func (r *SecretEngineMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var sem vaultv1alpha1.SecretEngineMount
	if err := r.Get(ctx, req.NamespacedName, &sem); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	client := r.VaultClient
	mounts, err := client.ListMounts()
	if err != nil {
		log.Error(err, "failed to list mounts")
		return ctrl.Result{}, err
	}

	statuses := make([]vaultv1alpha1.MountStatus, 0, len(mounts))

	for _, m := range sem.Spec.Mounts {
		mountPath := m.Path
		if !strings.HasSuffix(mountPath, "/") {
			mountPath += "/"
		}

		if _, exists := mounts[mountPath]; exists {
			log.Info("Secret engine already mounted", "mount", mountPath)
			statuses = append(statuses, vaultv1alpha1.MountStatus{
				Path:    m.Path,
				Mounted: true,
				Message: "Already mounted",
			})
			continue
		}

		err := client.MountSecretEngine(m.Path, m.Type, m.Options)
		if err != nil {
			log.Error(err, "failed to mount secret engine", "mount", mountPath)
			statuses = append(statuses, vaultv1alpha1.MountStatus{
				Path:    m.Path,
				Mounted: false,
				Message: "Failed to mount: " + err.Error(),
			})
			continue
		}

		log.Info("Secret engine mounted successfully", "mount", mountPath)
		statuses = append(statuses, vaultv1alpha1.MountStatus{
			Path:    m.Path,
			Mounted: true,
			Message: "Mounted successfully",
		})
	}

	sem.Status.Mounted = statuses

	if err := r.Status().Update(ctx, &sem); err != nil {
		log.Error(err, "failed to update SecretEngineMount status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretEngineMountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultv1alpha1.SecretEngineMount{}).
		Named("secretenginemount").
		Complete(r)
}
