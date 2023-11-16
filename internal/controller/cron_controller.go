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

// global variables declaration
// var secretName string
var configMapName string
var cronJobName string

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
	log := log.FromContext(ctx)

	cron := webappcronv1.Cron{}
	if err := r.Get(ctx, req.NamespacedName, &cron); err != nil {
		log.Error(err, "Unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(3).Info("Inside the controller")
	log.V(2).Info("Cron Config values", "Kind", cron.Kind, "Url", cron.Spec.Url)

	r.setGlobalVariables(&cron)

	// create the ConfigMap
	cfgMap := r.defineConfigMap(&cron)

	if err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: cron.Namespace}, cfgMap); err != nil {
		log.Error(err, "Unable to fetch CronJob")

		log.V(1).Info("Creating ConfigMap")
		cfgMap_error := r.Create(ctx, cfgMap)
		if cfgMap_error != nil {
			log.Error(cfgMap_error, "Unable to create ConfigMap for CronJob")
			return ctrl.Result{}, cfgMap_error
		}
		log.V(1).Info("ConfigMap created successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// cfgMap is present, check if anything needs to be updated
	// check Retries update
	if fmt.Sprint(cron.Spec.Retries) != cfgMap.Data["RETRIES"] {
		log.V(1).Info("Retries needs to be updated")
		cfgMap.Data["RETRIES"] = fmt.Sprint(cron.Spec.Retries)

		err := r.Update(ctx, cfgMap)
		if err != nil {
			log.Error(err, "Could not update the CronJob resource")
			return ctrl.Result{}, err
		}
		log.V(1).Info("CronJob updated successfully!")
	}

	//check URL update
	if fmt.Sprint(cron.Spec.Url) != cfgMap.Data["URL"] {
		log.V(1).Info("Url needs to be updated")
		cfgMap.Data["URL"] = fmt.Sprint(cron.Spec.Url)

		err := r.Update(ctx, cfgMap)
		if err != nil {
			log.Error(err, "Could not update the CronJob resource")
			return ctrl.Result{}, err
		}
		log.V(1).Info("CronJob updated successfully!")
	}

	// Create the CronJob
	job := r.defineCronJob(&cron)
	if err := r.Get(ctx, types.NamespacedName{Name: cronJobName, Namespace: cron.Namespace}, job); err != nil {
		log.V(1).Info("creating CronJob")
		err := r.Create(ctx, job)

		if err != nil {
			panic(err.Error())
		}
		log.V(2).Info("CronJob created", "req", req, "job", job)
		log.V(1).Info("CronJob created successfully!")
	}

	// TODO: Last successful CronJob completion
	if job.Status.LastSuccessfulTime != nil {
		cron.Status.LastSuccessfulTime = job.Status.LastSuccessfulTime
		err := r.Status().Update(ctx, &cron)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	// log.V(2).Info("The status of job is", "CronJob", job.Status.Active, "Last success", job.Status.LastSuccessfulTime

	// Finalizer
	cronFinalizer := "batch.tutorial.kubebuilder.io/finalizer"

	if cron.ObjectMeta.DeletionTimestamp.IsZero() {
		/*
		* The object is not being deleted, so if it does not have our finalizer,
		* then lets add the finalizer and update the object. This is equivalent
		* registering our finalizer.
		 */
		if !controllerutil.ContainsFinalizer(&cron, cronFinalizer) {
			// add Finalizer to CronJob if not present
			controllerutil.AddFinalizer(&cron, cronFinalizer)
			if err := r.Update(ctx, &cron); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The Cron object is being deleted
		if controllerutil.ContainsFinalizer(&cron, cronFinalizer) {

			if err := r.deleteExternalResources(&cron, job, cfgMap, ctx); err != nil {
				// if failed to delete the external dependency here, return with erro2
				// so that it can be retried
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&cron, cronFinalizer)
			if err := r.Update(ctx, &cron); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation since the item is being deleted
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// ConfigMap definition
func (r *CronReconciler) defineConfigMap(cron *webappcronv1.Cron) *apiv1.ConfigMap {
	cfgMap := apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: cron.Namespace,
		},
		// TODO: add more config data values as per API spec
		Data: map[string]string{
			"RETRIES":   fmt.Sprint(cron.Spec.Retries),
			"URL":       cron.Spec.Url,
			"BROKER_0":  cron.Spec.Broker_0,
			"BROKER_1":  cron.Spec.Broker_1,
			"BROKER_2":  cron.Spec.Broker_2,
			"CLIENT_ID": cron.Spec.Client_Id,
			"TOPIC":     cron.Spec.Topic,
		},
	}

	controllerutil.SetControllerReference(cron, &cfgMap, r.Scheme)
	return &cfgMap
}

// CronJob definition
func (r *CronReconciler) defineCronJob(cron *webappcronv1.Cron) *batchv1.CronJob {

	job := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: cron.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: apiv1.PodTemplateSpec{
						Spec: apiv1.PodSpec{
							Containers: []apiv1.Container{
								{
									Name:  "producer-app",
									Image: "sydrawat/producer",
									// Command: []string{"/bin/sh", "-c", "Hello from CronJob!"},
									EnvFrom: []apiv1.EnvFromSource{
										{
											ConfigMapRef: &apiv1.ConfigMapEnvSource{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: configMapName,
												},
											},
										},
										// TODO: add Secret resource configs
										// {
										// 	SecretRef: &apiv1.SecretEnvSource{
										// 		LocalObjectReference: apiv1.LocalObjectReference{
										// 			Name: secretName,
										// 		},
										// 	},
										// },
									},
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

// Delete the CronJob and its dependent resources
func (r *CronReconciler) deleteExternalResources(cron *webappcronv1.Cron, job *batchv1.CronJob, cfgMap *apiv1.ConfigMap, ctx context.Context) error {
	log := log.FromContext(ctx)
	if err := r.Get(ctx, types.NamespacedName{Name: cronJobName, Namespace: cron.Namespace}, job); err != nil {
		log.Error(err, "Cannot fetch the CronJob")
		return err
	}

	log.V(1).Info("CronJob found and ready to delete")
	err := r.Delete(ctx, job)
	if err != nil {
		return err
	}
	log.V(1).Info("Successfully deleted CronJob!")

	// Deleting CronJob ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: cron.Namespace}, cfgMap); err != nil {
		log.Error(err, "Cannot fetch the Config map")
		return err
	}
	err_cfgMap := r.Delete(ctx, cfgMap)

	if err_cfgMap != nil {
		return err_cfgMap
	}
	log.Info("ConfigMap deleted success!")
	return nil
}

// Define the global variables for ConfigMap, Secret & CronJob
func (r *CronReconciler) setGlobalVariables(cron *webappcronv1.Cron) {
	configMapName = cron.Name + "-config-map"
	// secretName = cron.Name + "-secret"
	cronJobName = cron.Name + "-cronjob"
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappcronv1.Cron{}).
		Owns(&batchv1.CronJob{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}
