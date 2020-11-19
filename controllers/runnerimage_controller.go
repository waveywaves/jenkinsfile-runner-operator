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

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/waveywaves/jenkinsfile-runner-operator/api/v1alpha1"
)

// RunnerImageReconciler reconciles a RunnerImage object
type RunnerImageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runnerimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runnerimages/status,verbs=get;update;patch

func (r *RunnerImageReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	runnerImageLogger := r.Log.WithValues("runnerimage", req.NamespacedName)

	// Fetch the Jenkinsfile Runner Image instance
	runnerImageInstance := &v1alpha1.RunnerImage{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerImageInstance)
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
	runnerImageLogger.Info("RunnerImage instance with name " + runnerImageInstance.Name + " found")

	return ctrl.Result{}, nil
}

func (r *RunnerImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RunnerImage{}).
		Complete(r)
}
