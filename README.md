# Vault Configurator

### Overview

`vault-configurator` is a Kubernetes operator built with [Operator SDK](https://sdk.operatorframework.io). It enables the declarative configuration of HashiCorp Vault, managing AuthMethods, SecretEngines, Policies, and VaultSecrets as Kubernetes resources. This allows you to configure Vault using GitOps workflows for consistency, automation, and version-controlled configurations.


## Features

- Declarative `AuthMethod` Management
  - Kubernetes auth method
  - TODO: JWT
  - TODO: Approle
  - TODO: OIDC
- Policy Automation:
  - Create and maintain vault policies declaratively, enabling version-controlled access management and persisted state.
- Secret Engine Creation:
  - Deploy and configure vault secret engines via custom resources for automated and consistent setup.
  - Supports v1 and v2 of KV secret engines
- Create Secrets:
  - Bootstrap secrets declaratively
  - Supports v1 and v2 of KV secret engines


## Install

```sh
helm repo add vault-configurator https://razahuss02.github.io/charts

helm upgrade --install vault-configurator vault-configurator/vault-configurator
```

### Bootstrap

After installing the vault-configurator operator in your cluster, you must configure [Kubernetes authentication](https://developer.hashicorp.com/vault/docs/auth/kubernetes) in Vault to allow the operator to communicate with your in-cluster Vault instance. 

You can set up this authentication manually, or use the provided bootstrap-auth.sh script, which:
* Enables the kubernetes auth method in vault.
* Deploys a temporary debug pod to retrieve a valid service account token.
* Configures Vault's kubernetes auth with the cluster endpoint and JWT from serviceaccount.
* Creates a vault policy granting required permissions.
* Creates a vault role bound to the operato's service account with the appropriate policies.
```sh
./bootstrap-auth.sh
```

> **Note:**
> 
> By default, the script configures a Vault policy with broad permissions, granting access to any secret engine mounts and their secrets that the operator manages.
> 
> It is strongly recommended to review and adjust this policy in the script to adhere to the principle of least privilege access for your environment.


### Creating custom resources

To see example definitions of how to create resources such as `AuthMethod`, `Policies`, `SecretEngineMount`, and `VaultSecret` refer to the [config/samples](config/samples/) directory in this repository.

