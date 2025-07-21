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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vaultv1alpha1 "github.com/apexdriver/vault-configurator/api/v1alpha1"
	vault "github.com/apexdriver/vault-configurator/internal/vault"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	VaultClient vault.VaultClient
}

func NewPolicyReconciler(mgr ctrl.Manager, vc vault.VaultClient) *PolicyReconciler {
	return &PolicyReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		VaultClient: vc,
	}
}

// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=policies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vault.apexdriver.dev,resources=policies/finalizers,verbs=update

func (r *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var policy vaultv1alpha1.Policy

	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	client := r.VaultClient

	for name, p := range policy.Spec.Policies {
		log.Info("Updating policy", "name", name)
		err := client.PutPolicy(name, p)
		if err != nil {
			log.Error(err, "failed to create policy", "name", name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultv1alpha1.Policy{}).
		Named("policy").
		Complete(r)
}
