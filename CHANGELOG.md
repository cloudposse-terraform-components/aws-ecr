# CHANGELOG

## 1.0.0

### Breaking Changes

- `image_names` has been removed in favor of `repository_name` - this component only supports a single repository.

  ```diff
  - image_names: [
  -  - "cloudposse-examples/app-on-ecs"
  -  - "cloudposse/foo"
  -  - "cloudposse/bar"
  - ]

  + component:
  +   ecr/repositories/app-on-ecs:
  +     metadata:
  +       component: ecr
  +       inherits:
  +         - ecr/defaults
  +     vars:
  +       repository_name: cloudposse-examples/app-on-ecs
  +       name: ecr-app-on-ecs
  ---
  +   ecr/repositories/foo:
  +     metadata:
  +       component: ecr
  +       inherits:
  +         - ecr/defaults
  +     vars:
  +       repository_name: cloudposse-examples/foo
  +       name: ecr-foo
  ---
  +   ecr/repositories/bar:
  +     metadata:
  +       component: ecr
  +       inherits:
  +         - ecr/defaults
  +     vars:
  +       repository_name: cloudposse-examples/bar
  +       name: ecr-bar
  ```

  The Cloudposse migration for example looked like:

  ```console
  . #stacks/catalog/ecr/
  └── repositories
      ├── app-on-ecs.yaml
      ├── app-on-eks-with-argocd.yaml
      ├── defaults.yaml
      ├── docker-component-build.yaml
      ├── example-app-on-ecs.yaml
      ├── example-monorepo.yaml
      ├── infra-live.yaml
      └── example-app-on-eks.yaml
  ```

  ```console
  # stacks/orgs/acme/core/us-east-2/artifacts.yaml
  imports:
  - catalog/ecr/*
  ```

  Defaults matches the README using the default values. and each repository entry is defined in it's own catalog entry.

- Lifecycle rules are now defined via the `lifecycle_rules` input.
