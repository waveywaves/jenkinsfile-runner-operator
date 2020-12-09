/*


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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/waveywaves/jenkinsfile-runner-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RunReconciler reconciles a Run object
type RunReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var runLogger logr.Logger

// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runs/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *RunReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	// Initialize Logger
	runLogger = r.Log.WithValues("runnerImage", req.NamespacedName)

	// Fetch the RunnerImage instance
	runInstance := &v1alpha1.Run{}
	err := r.Client.Get(ctx, req.NamespacedName, runInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	runLogger.Info("Run with name " + runInstance.Name + " detected")
	//runnerImageInstance.Status.Phase = PhaseInitialized
	//err = r.Status().Update(context.TODO(), runnerImageInstance)
	//if err != nil {
	//	return ctrl.Result{}, err
	//}

	jfConfigMapRef := runInstance.Spec.Jenkinsfile.ConfigMapRef
	cascConfigMapRef := runInstance.Spec.ConfigurationAsCode.ConfigMapRef
	volumes := []corev1.Volume{
		{
			Name: "jenkinsfile",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: jfConfigMapRef},
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "jenkinsfile",
			MountPath: "/workspace/jenkinsfile",
			SubPath:   "Jenkinsfile",
		},
	}

	if len(cascConfigMapRef) > 0 {
		cascVolume := corev1.Volume{
			Name: "casc",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cascConfigMapRef},
				},
			},
		}
		cascVolumeMount := corev1.VolumeMount{
			Name:      "casc",
			MountPath: "/usr/share/jenkins/ref/casc",
		}
		volumes = append(volumes, cascVolume)
		volumeMounts = append(volumeMounts, cascVolumeMount)
	}

	runPod := &corev1.Pod{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      fmt.Sprintf("jfr-run-%s", runInstance.Name),
			Namespace: req.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "jfr-run",
					Image:           runInstance.Spec.Image,
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: "Always",
				},
			},
			Volumes:       volumes,
			RestartPolicy: "Never",
		},
	}
	controllerutil.SetControllerReference(runInstance, runPod, r.Scheme)
	err = r.Client.Create(context.TODO(), runPod)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Run{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
