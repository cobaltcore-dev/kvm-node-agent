<!--
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors

SPDX-License-Identifier: Apache-2.0
-->
# kvm-node-agent [![REUSE status](https://api.reuse.software/badge/github.com/cobaltcore-dev/kvm-node-agent)](https://api.reuse.software/info/github.com/cobaltcore-dev/kvm-node-agent) [![Checks](https://github.com/cobaltcore-dev/kvm-node-agent/actions/workflows/checks.yaml/badge.svg)](https://github.com/cobaltcore-dev/kvm-node-agent/actions/workflows/checks.yaml)

KVM Node agent for controlling Hypervisor objects via Kubernetes API.

## Description

The KVM node agent is a kubernetes operator that runs on every KVM node provides introspection to KVM specif services and status.

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Building

To build the KVM node agent binary, run:

```bash
make build-all
```

The compiled binary will be available at `build/manager`.

To install the binary to your system:

```bash
make install
```

### Installing CRDs

To install the Custom Resource Definitions (CRDs) into your Kubernetes cluster:

```bash
make install-crds
```

This command generates the necessary CRD manifests and applies them to the cluster specified in your `~/.kube/config`. The CRDs define the `Hypervisor` and `Migration` custom resources that the KVM node agent uses to manage KVM instances.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/cobaltcore-dev/kvm-node-agent/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/cobaltcore-dev/kvm-node-agent/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## License

Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Please see our [LICENSE](LICENSE) for copyright and license information.
Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/cobaltcore-dev/kvm-node-agent).
