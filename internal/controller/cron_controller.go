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
	b64 "encoding/base64"
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

// Global variables

// var secretName string
var robocop string
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
//+kubebuilder:rbac:groups="batch",resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=cronjobs/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the Cron object against the actual cluster state, and then
// performs operations to make the cluster state reflect the state specified by
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

	r.setGlobalVariables(&cron)

	// CREATE: ConfigMap
	cfgMap := r.defineConfigMap(&cron)

	if err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: cron.Namespace}, cfgMap); err != nil {
		log.Error(err, "ConfigMap not created for CronJob")

		log.V(1).Info("Creating ConfigMap")
		cfgMap_error := r.Create(ctx, cfgMap)
		if cfgMap_error != nil {
			log.Error(cfgMap_error, "Unable to create ConfigMap for CronJob")
			return ctrl.Result{}, cfgMap_error
		}
		log.V(1).Info("ConfigMap created successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// CREATE: ImagePullSecrets
	scrt := r.defineQuaySecret(&cron)
	if err := r.Get(ctx, types.NamespacedName{Name: robocop, Namespace: cron.Namespace}, scrt); err != nil {
		log.Error(err, "Secret not created for CronJob")

		log.V(1).Info("Creating Secret")
		scrt_error := r.Create(ctx, scrt)
		if scrt_error != nil {
			log.Error(scrt_error, "Unable to create Secret for CronJob")
			return ctrl.Result{}, scrt_error
		}
		log.V(1).Info("Secret created successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// CREATE: CronJob
	job := r.defineCronJob(&cron)
	if err := r.Get(ctx, types.NamespacedName{Name: cronJobName, Namespace: cron.Namespace}, job); err != nil {
		log.V(1).Info("Creating CronJob")
		err := r.Create(ctx, job)

		if err != nil {
			panic(err.Error())
		}
		log.V(2).Info("CronJob created", "req", req, "job", job)
		log.V(1).Info("CronJob created successfully!")
	}

	// Last Scheduled CronJob time
	if job.Status.LastScheduleTime != nil {
		cron.Status.LastScheduleTime = job.Status.LastScheduleTime
		err := r.Client.Status().Update(ctx, &cron)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("CronJob LastScheduleTime updated")
	}

	// CronJob Active Status
	if job.Status.Active != nil {
		cron.Status.Active = true
		err := r.Status().Update(ctx, &cron)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("CronJob Status updated")
	}

	// UPDATE: Reconcile cfgMap
	if fmt.Sprint(cron.Spec.Schedule) != cfgMap.Data["SCHEDULE"] {
		log.V(1).Info("Updating Schedule")
		cfgMap.Data["SCHEDULE"] = fmt.Sprint(cron.Spec.Schedule)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if cron.Spec.Name != cfgMap.Data["NAME"] {
		log.V(1).Info("Updating Name")
		cfgMap.Data["NAME"] = fmt.Sprint(cron.Spec.Name)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Retries) != cfgMap.Data["RETRIES"] {
		log.V(1).Info("Updating Retries")
		cfgMap.Data["RETRIES"] = fmt.Sprint(cron.Spec.Retries)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Response) != cfgMap.Data["RESPONSE"] {
		log.V(1).Info("Updating Response")
		cfgMap.Data["RESPONSE"] = fmt.Sprint(cron.Spec.Response)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Url) != cfgMap.Data["URL"] {
		log.V(1).Info("Updating Url")
		cfgMap.Data["URL"] = fmt.Sprint(cron.Spec.Url)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Broker_0) != cfgMap.Data["BROKER_0"] {
		log.V(1).Info("Updating Broker_0")
		cfgMap.Data["BROKER_0"] = fmt.Sprint(cron.Spec.Broker_0)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Broker_1) != cfgMap.Data["BROKER_1"] {
		log.V(1).Info("Updating Broker_1")
		cfgMap.Data["BROKER_1"] = fmt.Sprint(cron.Spec.Broker_1)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Broker_2) != cfgMap.Data["BROKER_2"] {
		log.V(1).Info("Updating Broker_2")
		cfgMap.Data["BROKER_2"] = fmt.Sprint(cron.Spec.Broker_2)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Client_Id) != cfgMap.Data["CLIENT_ID"] {
		log.V(1).Info("Updating Client ID")
		cfgMap.Data["CLIENT_ID"] = fmt.Sprint(cron.Spec.Client_Id)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.Topic) != cfgMap.Data["TOPIC"] {
		log.V(1).Info("Updating Topic")
		cfgMap.Data["TOPIC"] = fmt.Sprint(cron.Spec.Topic)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	// TODO: update secret data if changed
	// if fmt.Sprint(cron.Spec.DockerConfigJSON) != string(scrt.Data["DOCKERCONFIGJSON"]) {
	// 	log.V(1).Info("Updating DockerConfigJSON")
	// 	scrt.Data["DOCKERCONFIGJSON"] = cron.Spec.DockerConfigJSON
	// 	_, err := r.UpdateSecret(ctx, scrt, &cron, req)
	// 	return ctrl.Result{}, err
	// }

	if fmt.Sprint(cron.Spec.FailedJobsHistoryLimit) != cfgMap.Data["FAILED_JOBS_HISTORY_LIMIT"] {
		log.V(1).Info("Updating FailedJobsHistoryLimit")
		cfgMap.Data["FAILED_JOBS_HISTORY_LIMIT"] = fmt.Sprint(cron.Spec.FailedJobsHistoryLimit)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	if fmt.Sprint(cron.Spec.SuccessfulJobsHistoryLimit) != cfgMap.Data["SUCCESSFUL_JOBS_HISTORY_LIMIT"] {
		log.V(1).Info("Updating SuccessfulJobsHistoryLimit")
		cfgMap.Data["SUCCESSFUL_JOBS_HISTORY_LIMIT"] = fmt.Sprint(cron.Spec.SuccessfulJobsHistoryLimit)
		_, err := r.UpdateCfgMap(ctx, cfgMap, &cron, req)
		return ctrl.Result{}, err
	}

	// FINALIZER
	cronFinalizer := "batch.tutorial.kubebuilder.io/finalizer"

	if cron.ObjectMeta.DeletionTimestamp.IsZero() {
		/*
		* The object is not being deleted, so if it does not have our finalizer,
		* then lets add the finalizer and update the object. This is equivalent
		* registering our finalizer.
		 */
		if !controllerutil.ContainsFinalizer(&cron, cronFinalizer) {
			// Add Finalizer to CronJob if not present
			controllerutil.AddFinalizer(&cron, cronFinalizer)
			if err := r.Update(ctx, &cron); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Delete Cron object
		if controllerutil.ContainsFinalizer(&cron, cronFinalizer) {

			if err := r.deleteExternalResources(&cron, job, cfgMap, scrt, ctx); err != nil {
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
		Data: map[string]string{
			"SCHEDULE":                      cron.Spec.Schedule,
			"NAME":                          cron.Spec.Name,
			"URL":                           cron.Spec.Url,
			"RETRIES":                       fmt.Sprint(cron.Spec.Retries),
			"RES_CODE":                      fmt.Sprint(cron.Spec.Response),
			"HTTP_CHECK_ID":                 fmt.Sprint(cron.Spec.HTTP_Check_Id),
			"BROKER_0":                      cron.Spec.Broker_0,
			"BROKER_1":                      cron.Spec.Broker_1,
			"BROKER_2":                      cron.Spec.Broker_2,
			"CLIENT_ID":                     cron.Spec.Client_Id,
			"TOPIC":                         cron.Spec.Topic,
			"FAILED_JOBS_HISTORY_LIMIT":     fmt.Sprint(cron.Spec.FailedJobsHistoryLimit),
			"SUCCESSFUL_JOBS_HISTORY_LIMIT": fmt.Sprint(cron.Spec.SuccessfulJobsHistoryLimit),
		},
	}

	controllerutil.SetControllerReference(cron, &cfgMap, r.Scheme)
	return &cfgMap
}

// Secret definition
func (r *CronReconciler) defineQuaySecret(cron *webappcronv1.Cron) *apiv1.Secret {
	decodedValue, _ := b64.StdEncoding.DecodeString(cron.Spec.DockerConfigJSON)
	scrt := apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      robocop,
			Namespace: cron.Namespace,
		},
		Type: apiv1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			".dockerconfigjson": string(decodedValue),
		},
	}

	controllerutil.SetControllerReference(cron, &scrt, r.Scheme)
	return &scrt
}

// CronJob definition
func (r *CronReconciler) defineCronJob(cron *webappcronv1.Cron) *batchv1.CronJob {

	job := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: cron.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			Schedule:                   cron.Spec.Schedule,
			SuccessfulJobsHistoryLimit: &cron.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     &cron.Spec.FailedJobsHistoryLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: &cron.Spec.Retries,
					Template: apiv1.PodTemplateSpec{
						Spec: apiv1.PodSpec{
							ImagePullSecrets: []apiv1.LocalObjectReference{
								{
									Name: robocop,
								},
							},
							Containers: []apiv1.Container{
								{
									Name:            "producer-app",
									Image:           "quay.io/pwncorp/producer:latest",
									ImagePullPolicy: apiv1.PullAlways,
									//Command: []string{"/bin/sh", "-c", "date; echo Hello from the Kubernetes cluster"},
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
func (r *CronReconciler) deleteExternalResources(cron *webappcronv1.Cron, job *batchv1.CronJob, cfgMap *apiv1.ConfigMap, scrt *apiv1.Secret, ctx context.Context) error {
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

	// Deleting CronJob Secret
	if err := r.Get(ctx, types.NamespacedName{Name: robocop, Namespace: cron.Namespace}, scrt); err != nil {
		log.Error(err, "Cannot fetch the Secret")
		return err
	}
	err_scrt := r.Delete(ctx, scrt)

	if err_scrt != nil {
		return err_scrt
	}
	log.Info("Secret deleted success!")
	return nil
}

// Update function for configMap updates
func (r *CronReconciler) UpdateCfgMap(ctx context.Context, cfgMap *apiv1.ConfigMap, cron *webappcronv1.Cron, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	err := r.Update(ctx, cfgMap)
	if err != nil {
		log.Error(err, "Could not update the CronJob ConfigMap resource")
		return ctrl.Result{}, err
	}
	log.V(1).Info("CronJob ConfigMap updated successfully!")
	return ctrl.Result{}, nil
}

// TODO: Update function for secret updates
func (r *CronReconciler) UpdateSecret(ctx context.Context, scrt *apiv1.Secret, cron *webappcronv1.Cron, req ctrl.Request) (ctrl.Request, error) {
	log := log.FromContext(ctx)
	err := r.Update(ctx, scrt)
	if err != nil {
		log.Error(err, "Could not update the CronJon Secret resource")
		return ctrl.Request{}, err
	}
	log.V(1).Info("CronJob Secret updated successfully!")
	return ctrl.Request{}, nil
}

// Define the global variables for ConfigMap, Secret & CronJob
func (r *CronReconciler) setGlobalVariables(cron *webappcronv1.Cron) {
	configMapName = cron.Name + "-config-map"
	// secretName = cron.Name + "-secret"
	robocop = cron.Name + "-quay"
	cronJobName = cron.Name + "-cronjob"
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappcronv1.Cron{}).
		Owns(&batchv1.CronJob{}).
		Owns(&apiv1.ConfigMap{}).
		Owns(&apiv1.Secret{}).
		Complete(r)
}
