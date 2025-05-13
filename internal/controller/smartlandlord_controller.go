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
    "fmt"
	"time"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilerrors "k8s.io/apimachinery/pkg/util/errors"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"
//    "sigs.k8s.io/controller-runtime/pkg/reconcile"

	realtortoolsv1alpha1 "github.com/nowelbliss2/smartlandlord-operator/api/v1alpha1"
)

// SmartlandlordReconciler reconciles a Smartlandlord object
type SmartlandlordReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=realtortools.realtordevelopments.io,resources=smartlandlords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=realtortools.realtordevelopments.io,resources=smartlandlords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=realtortools.realtordevelopments.io,resources=smartlandlords/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Smartlandlord object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *SmartlandlordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

    smartlandlord := &realtortoolsv1alpha1.Smartlandlord{}
    err := r.Get(ctx, req.NamespacedName, smartlandlord)
    if err != nil && errors.IsNotFound(err) {
        logger.Info("operator resource object not found")
        return ctrl.Result{}, nil
    } else if err != nil {
          logger.Error(err, "error getting operator resource object")
          updateStatus(smartlandlord, "OperatorSucceeded", "OperatorResourceNotAvailable", fmt.Sprintf("error getting operator resource object: %v", err), metav1.ConditionFalse)
          return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, smartlandlord)})
    } 
    
	replicas := smartlandlord.Spec.Replicas

    smartlandlordDeployment := &appsv1.Deployment{}
	expectedSmartlandlordDeployment := r.defineDeployment(smartlandlord)
    err = r.Get(ctx, types.NamespacedName{Name: smartlandlord.Name, Namespace: smartlandlord.Namespace}, smartlandlordDeployment)
    if err != nil && errors.IsNotFound(err) {
        logger.Info("Smartlandlord deployment not found. Creating or re-creating service")
        err := r.Create(ctx, expectedSmartlandlordDeployment)
	    if err != nil {
            logger.Error(err, fmt.Sprintf("error creating %s Smartlandlord deployment.", req.Name))
            updateStatus(smartlandlord, "OperatorFailed", "SmartlandlordCreationFailed", fmt.Sprintf("error creating %s deployment: %v", req.Name, err), metav1.ConditionFalse)
            return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, smartlandlord)})

		}

		logger.Info(fmt.Sprintf("Deployment %s created", req.Name))

    } else {
      
        var current_replicas int32 = *smartlandlordDeployment.Spec.Replicas
	    var expected_replicas int32 = smartlandlord.Spec.Replicas
	
	    if current_replicas != expected_replicas {
	        smartlandlordDeployment.Spec.Replicas = &replicas
	        if err = r.Update(ctx, smartlandlordDeployment); err != nil {
                logger.Error(err, "Failed to update Deployment",
                    "Deployment.Namespace", smartlandlord.Namespace, "Deployment.Name", smartlandlord.Name)

            return ctrl.Result{}, err
            }

	    }

    }
    
	smartlandlordService := &corev1.Service{}
    expectedSmartlandlordService := r.defineService(smartlandlord)
    err = r.Get(ctx, types.NamespacedName{Name: smartlandlord.Name, Namespace: smartlandlord.Namespace}, smartlandlordService)
    if err != nil && errors.IsNotFound(err) {
        logger.Info("Service resource smartlandlord-service not found. Creating or re-creating service")
		err = r.Create(ctx, expectedSmartlandlordService)
	    if err != nil {
	        logger.Error(err, fmt.Sprintf("error creating %s web service.", smartlandlord.Name))
            updateStatus(smartlandlord, "OperatorFailed", "WebServiceCreationFailed", fmt.Sprintf("error creating web service: %s", smartlandlord.Name, err), metav1.ConditionFalse)
            return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, smartlandlord)})
		
	    }
        logger.Info("web service smartlandlord-service created")

	}


    return ctrl.Result{}, nil  
}


func updateStatus(w *realtortoolsv1alpha1.Smartlandlord, statusType string, statusReason string, statusMessage string, condition metav1.ConditionStatus) {
	meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
		Type:               statusType,
		Status:             condition,
		Reason:             statusReason,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Message:            statusMessage,
	})
}


func (r *SmartlandlordReconciler) defineDeployment(smartlandlord *realtortoolsv1alpha1.Smartlandlord) *appsv1.Deployment {
    
    replicas := smartlandlord.Spec.Replicas

    smartlandlordDeployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      smartlandlord.Name,
            Namespace: smartlandlord.Namespace,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app":  "smartlandlord",
                    "name": smartlandlord.Name,
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":  "smartlandlord",
                        "name": smartlandlord.Name,
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "smartlandlord-core",
                            Image: smartlandlord.Spec.Image,
                        },
                    },
                },
            },
        },
    }
    ctrl.SetControllerReference(smartlandlord, smartlandlordDeployment, r.Scheme)
    return smartlandlordDeployment
}



func (r *SmartlandlordReconciler) defineService(smartlandlord *realtortoolsv1alpha1.Smartlandlord) *corev1.Service {

    smartlandlordService := &corev1.Service {
        ObjectMeta: metav1.ObjectMeta {
            Name: smartlandlord.Name,
            Namespace: smartlandlord.Namespace,
        },
        Spec: corev1.ServiceSpec{
            Selector: map[string]string{
                "app": "smartlandlord",
                "name": smartlandlord.Name,
            },
            Ports: []corev1.ServicePort{
                {
                    Name: "http",
                    Port: 8000,
                    Protocol: "TCP",
                },
            },
        },
    }

    ctrl.SetControllerReference(smartlandlord, smartlandlordService, r.Scheme)
	return smartlandlordService
}

// SetupWithManager sets up the controller with the Manager.
func (r *SmartlandlordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&realtortoolsv1alpha1.Smartlandlord{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
