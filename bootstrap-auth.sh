#!/bin/bash

set -euo pipefail

SERVICEACCOUNT_NAME="vault-configurator"
VAULT_POLICY_NAME="vault-configurator"
VAULT_ROLE_NAME="vault-configurator"

echo "[INFO] Enabling Kubernetes auth method in Vault..."
vault auth enable kubernetes || echo "[INFO] Kubernetes auth already enabled."

echo "[INFO] Deploying debug pod to retrieve projected service account token..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: vault-config-debug
  namespace: default
spec:
  serviceAccountName: $SERVICEACCOUNT_NAME
  containers:
  - name: debug
    image: busybox
    command: ["sleep", "3600"]
EOF

echo "[INFO] Waiting for debug pod to be ready..."
kubectl wait --for=condition=Ready pod/vault-config-debug --timeout=60s

echo "[INFO] Retrieving JWT token from debug pod..."
JWT_TOKEN=$(kubectl exec -it vault-config-debug -- cat /var/run/secrets/kubernetes.io/serviceaccount/token | tr -d '\r')

echo "[INFO] Configuring Vault Kubernetes auth with retrieved token..."
vault write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc" \
  kubernetes_ca_cert=@<(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d) \
  token_reviewer_jwt="$JWT_TOKEN"

echo "[INFO] Deleting debug pod..."
kubectl delete pod vault-config-debug

echo "[INFO] Creating '$VAULT_POLICY_NAME' policy in Vault..."
vault policy write $VAULT_POLICY_NAME - <<EOF
path "*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
EOF

echo "[INFO] Creating role 'vault-configurator' bound to the service account..."
vault write auth/kubernetes/role/$VAULT_ROLE_NAME \
  bound_service_account_names=$SERVICEACCOUNT_NAME \
  bound_service_account_namespaces="*" \
  policies=$VAULT_POLICY_NAME \
  ttl=1h

echo "[INFO] Vault Kubernetes auth bootstrap complete."
