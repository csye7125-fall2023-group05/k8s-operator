/*
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
*/

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappcronv1 "csye7125-fall2023-group05.cloud/cron/api/v1"
	batchv1 "k8s.io/api/batch/v1"

	apiv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CronReconciler reconciles a Cron object
type CronReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webappcron.csye7125-fall2023-group05.cloud,resources=crons,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webappcron.csye7125-fall2023-group05.cloud,resources=crons/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webappcron.csye7125-fall2023-group05.cloud,resources=crons/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cron object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *CronReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	cron := webappcronv1.Cron{}
	if err := r.Get(ctx, req.NamespacedName, &cron); err != nil {
		l.Error(err, "unable to fetch CronJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Inside the controller")
	l.Info("Cron Config values", "Kind", cron.Kind, "Url", cron.Spec.Url)

	//retries := cron.Spec.Retries

	cm := r.defineConfigMap(&cron)

	if err := r.Get(ctx, types.NamespacedName{Name: "cron-config-map", Namespace: "default"}, cm); err != nil {
		l.Error(err, "unable to fetch CronJob")

		l.Info("creating config map")
		cm_error := r.Create(ctx, cm)

		if cm_error != nil {
			l.Error(cm_error, "Unable to create the config map for cron job")
			return ctrl.Result{}, cm_error
		}

		l.Info("ConfigMap created success!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// here cm is there and now check if anything needs to be updated ?
	if fmt.Sprint(cron.Spec.Retries) != cm.Data["retries"] {
		l.Info("retries need to be updated")

		cm.Data["retries"] = fmt.Sprint(cron.Spec.Retries)

		err := r.Update(ctx, cm)

		if err != nil {
			l.Error(err, "Could not update the CronJob resource")
			return ctrl.Result{}, err
		}

		l.Info("Update of the CronJob Success!")
	}

	job := r.defineCronJob(&cron)

	if err := r.Get(ctx, types.NamespacedName{Name: cron.Name, Namespace: "default"}, job); err != nil {

		l.Info("creating cron job")
		err := r.Create(ctx, job)

		if err != nil {
			panic(err.Error())
		}

		l.Info("CronJob created", "req", req, "job", job)
		l.Info("CronJob created success!")

	}
	if job.Status.LastSuccessfulTime != nil {
		cron.Status.LastSuccessfulTime = job.Status.LastSuccessfulTime
		err := r.Status().Update(ctx, &cron)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	//l.Info("The status of job is", "CronJob", job.Status.Active, "Last success", job.Status.LastSuccessfulTime

	// end of finalizer

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappcronv1.Cron{}).
		Owns(&batchv1.CronJob{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}

// new

// function for ConfigMap
func (r *CronReconciler) defineConfigMap(cron *webappcronv1.Cron) *apiv1.ConfigMap {
	retries := cron.Spec.Retries
	cm := apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cron-config-map",
			Namespace: "default",
		},
		Data: map[string]string{
			"retries": fmt.Sprint(retries),
		},
	}
	controllerutil.SetControllerReference(cron, &cm, r.Scheme)
	return &cm
}

// function for CronJob
func (r *CronReconciler) defineCronJob(cron *webappcronv1.Cron) *batchv1.CronJob {

	job := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{

		Name:      cron.Name,
		Namespace: "default"},

		Spec: batchv1.CronJobSpec{

			Schedule: "* * * * *",

			JobTemplate: batchv1.JobTemplateSpec{

				Spec: batchv1.JobSpec{

					Template: apiv1.PodTemplateSpec{

						Spec: apiv1.PodSpec{

							Containers: []apiv1.Container{

								{

									Name: "hello",

									Image: "busybox:1.28",

									Command: []string{"/bin/sh", "-c", "date; echo Hello from the Kubernetes cluster"},
								},
							},

							RestartPolicy: apiv1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(cron, job, r.Scheme)
	return job
}
