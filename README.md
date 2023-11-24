# k8s-operator

This will help you understand how to write `Operators` using `Kubebuilder` SDK which will help create a Custom Resource Definition, Custom Resource, Controllers and Reconciliation logic.
Basically, Reconcile the `current` state to the `desired` state

## Description

This is a custom kubernetes operator which creates a Cron Custom Resource Definition, based on which the controller watches for a Custom Resource and applies the reconciliation logic using the `Reconciler` in the operator controller.

The controller only creates, watches and deletes the `Cron` resource, and updates the `ConfigMap` and `Secret` resources.

## Getting Started

### Prerequisites

- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
# If NS is not specified, operator will watch the default namespace
make docker-build docker-push IMG=<some-registry>/k8s-operator:tag NS=<your-namespace>
```

> **NOTE:** This image ought to be published in the personal registry you specified.
> And it is required to have access to pull the image from the working environment.
> Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/k8s-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
> privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

> **NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

**To run the controller based on a specific namespace**

```sh
NAMESPACE=<ns_name> make run
```

## :mechanical_arm: Operator

- S/w that runs within k8s
- eventual consistency which it will get in the desired state what was expected
- Any changes you do in type.go file run the `make manifests` command

```bash
# any changes to types file run the below command
make manifests
```

## :brain: Kubebuilder

- After bootstrapping, you will notice a set of generated directories and files. Each has its specific purpose:

  `Dockerfile`: A Dockerfile used for building a containerized version of the operator.

  `Makefile`: A Makefile containing targets for building, testing, and deploying the operator.

  `PROJECT`: A YAML file containing project metadata.

  `api/`: A directory that will contain the API definitions for our CRDs.

  `internal/controllers/`: A directory that will contain the controller implementations.

  `config/`: A directory that contains different configuration on how your operator would be deployed and interact with k8s cluster

  `cmd/main.go`: The entrypoint of our operator.

## :microbe: CR and CRD

- Think of `CRDs` as the definition of our **custom Objects**, while `CRs` are **instances** of them.

## :seedling: Sample which the Kubebuilder provides

- sample is a folder which provides a sample custom resource i.e an object ( YAML file)
- This can be used to edit it as per the req to work with in cluster

## :jigsaw: Schema

- `TypeMeta` (which describes API version and Kind), and also contains `ObjectMeta`, which holds things like name, namespace, and labels.

## :traffic_light: Controller

- Each controller implements a control loop that watches the cluster's shared state via the API server and makes changes needed, consistently bringing it back to the desired state.
- Operators’ controllers run in the worker nodes.

## :yo_yo: Reconciler

- In every controller, the reconciler is the logic that’s triggered by cluster events. The reconcile function takes the name of an object and returns whether or not the state matches the desired state.
- Need to implement the logic in the internal/controller/operator_controller.go file in the function `Reconcile()`

> **NOTE**: `make install` will install our custom resource in the k8s cluster and `make run` will connect to the k8s cluster

## :dart: Validation & Markers

- markers, such as `+kubebuilder:validation:Minimum=1`
- These markers help in defining validations and criteria, ensuring that data provided by users — when they create or edit a Custom Resource for the **XYZ Kind** is properly validated.

```bash
+kubebuilder:validation:Minimum=1
+kubebuilder:validation:Maximum=3
```

## Some important points

- How to watch pods
- SetupWithManager -> what custom resource definition to watch out for this custom controller
- Everytime the CR is created or updated the Custom Controller is notified
- And it is notified in the Reconciler function just above the SetupWithManager function
- `req` holds a namespaced name and thats all it has

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

> **NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
