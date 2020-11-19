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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/waveywaves/jenkinsfile-runner-operator/api/v1alpha1"
)

const (
	RunnerContainerName = "runner"
	RunnerImage         = "jenkins/jenkinsfile-runner:latest"
)

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runners/status,verbs=get;update;patch

func (r *RunnerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	runnerLogger := r.Log.WithValues("runner", req.NamespacedName)

	// Fetch the Jenkinsfile Runner instance
	runnerInstance := &v1alpha1.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	runnerLogger.Info("Runner instance with name " + runnerInstance.Name + " found")

	runnerPod := r.getRunnerPod(req)
	err = r.Client.Create(context.TODO(), runnerPod)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) getRunnerPod(req ctrl.Request) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRunnerPodName(req.Name),
			Namespace: req.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            RunnerContainerName,
					Image:           RunnerImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c", "--"},
					Args:            []string{"while true; do sleep 30; done;"},
					Stdin:           true,
					TTY:             true,
				},
			},
		},
	}
}

func getRunnerPodName(runnerName string) string {
	return "jenkinsfile-runner-" + runnerName
}

func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Runner{}).
		Complete(r)
}
