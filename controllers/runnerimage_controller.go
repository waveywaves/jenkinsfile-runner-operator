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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/waveywaves/jenkinsfile-runner-operator/api/v1alpha1"
)

var (
	DockerfileStorageSuffix = "dockerfile-storage"
	DockerfileName          = "Dockerfile"
	DockerfileTemplate      = `FROM jenkins/jenkinsfile-runner
RUN cd /app/jenkins && jar -cvf jenkins.war *
RUN java -jar /app/bin/jenkins-plugin-manager.jar --war /app/jenkins/jenkins.war --plugin-file /usr/share/jenkins/ref/plugins.txt && rm /app/jenkins/jenkins.war
ENTRYPOINT /app/bin/jenkinsfile-runner-launcher -f /workspace/jenkinsfile/
`

	// Phases
	PhaseInitialized = "Initialized"
	PhaseStarted     = "Started"
	PhaseCompleted   = "Completed"
	PhaseError       = "Error"
)

var runnerImageLogger logr.Logger

// RunnerImageReconciler reconciles a RunnerImage object
type RunnerImageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runnerimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runnerimages/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkinsfilerunner.io,resources=runnerimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *RunnerImageReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	// Initialize Logger
	runnerImageLogger = r.Log.WithValues("runnerImage", req.NamespacedName)

	// Fetch the RunnerImage instance
	runnerImageInstance := &v1alpha1.RunnerImage{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerImageInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	runnerImageLogger.Info("Jenkins RunnerImage with name " + runnerImageInstance.Name + " detected")
	runnerImageInstance.Status.Phase = PhaseInitialized
	err = r.Status().Update(context.TODO(), runnerImageInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Initialize the RunnerImage
	err = r.InitResources(req, runnerImageInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RunnerImageReconciler) InitResources(req ctrl.Request, runnerImageInstance *v1alpha1.RunnerImage) error {
	err := r.initResources(req)
	if err != nil {
		return err
	}
	runnerImageInstance.Status.Phase = PhaseInitialized
	err = r.Status().Update(context.TODO(), runnerImageInstance)
	if err != nil {
		return err
	}
	return nil
}

func (r *RunnerImageReconciler) initResources(req ctrl.Request) error {
	// Define a new ConfigMap containing the Dockerfile used to build the image
	dockerfileConfigMap := &corev1.ConfigMap{}
	dfConfigMapNamespacedName := types.NamespacedName{Name: r.getJFRDockerfileConfigMapName(req.NamespacedName), Namespace: req.Namespace}
	err := r.Client.Get(context.TODO(), dfConfigMapNamespacedName, dockerfileConfigMap)
	if apierrors.IsNotFound(err) {
		runnerImageLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", dfConfigMapNamespacedName.Namespace, "ConfigMap.Name", dfConfigMapNamespacedName.Name)
		dockerfileConfigMap = r.getJFRDockerfile(req.NamespacedName)
		err = r.Client.Create(context.TODO(), dockerfileConfigMap)
		if err != nil {
			return err
		}
		// ConfigMap created successfully - don't requeue
	}
	runnerImageLogger.Info("ConfigMap exists", "ConfigMap.Namespace", dockerfileConfigMap.Namespace, "ConfigMap.Name", dockerfileConfigMap.Name)

	kanikoBuildPod, err := r.GetKanikoPod(req)
	if apierrors.IsNotFound(err) {
		runnerImageLogger.Info("Creating a new Pod", "Pod.Namespace", kanikoBuildPod.Namespace, "ConfigMap.Name", kanikoBuildPod.Name)
		kanikoBuildPod, err = r.CreateKanikoPod(req)
		if err != nil {
			return err
		}
		// Pod created successfully - don't requeue
	}
	runnerImageLogger.Info("Kaniko Build Pod exists", "Pod.Namespace", kanikoBuildPod.Namespace, "Pod.Name", kanikoBuildPod.Name)

	return nil
}

func (r *RunnerImageReconciler) CreateKanikoPod(req ctrl.Request) (*corev1.Pod, error) {
	kanikoBuildPod := r.getKanikoPodDefinition(req.NamespacedName)
	err := r.Client.Create(context.TODO(), kanikoBuildPod)
	return kanikoBuildPod, err
}

func (r *RunnerImageReconciler) GetKanikoPod(req ctrl.Request) (*corev1.Pod, error) {
	kanikoBuildPod := &corev1.Pod{}
	kPodNamespacedName := types.NamespacedName{Name: r.getKanikoPodName(req.NamespacedName), Namespace: req.Namespace}
	err := r.Client.Get(context.TODO(), kPodNamespacedName, kanikoBuildPod)
	return kanikoBuildPod, err
}

func (r *RunnerImageReconciler) getJFRDockerfile(nn types.NamespacedName) *corev1.ConfigMap {
	data := map[string]string{
		DockerfileName: DockerfileTemplate,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getJFRDockerfileConfigMapName(nn),
			Namespace: nn.Namespace,
		},
		Data: data,
	}
}

func (r *RunnerImageReconciler) getKanikoPodDefinition(nn types.NamespacedName) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getKanikoPodName(nn),
			Namespace: nn.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "jfr-build",
					Image: "gcr.io/kaniko-project/executor:corev1.3.0",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      fmt.Sprintf("%s-%s", nn.Name, DockerfileStorageSuffix),
							MountPath: "/workspace",
						},
						{
							Name:      r.getJFRDockerfileConfigMapName(nn),
							MountPath: "/dockerfile",
						},
					},
					Args: []string{
						"--dockerfile=/dockerfile/Dockerfile",
						"--context=dir://workspace/",
						"--no-push",
						"--digest-file=/dev/termination-log",
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: r.getJFRDockerfileConfigMapName(nn),
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: r.getJFRDockerfileConfigMapName(nn)},
						},
					},
				},
				{
					Name: fmt.Sprintf("%s-%s", nn.Name, DockerfileStorageSuffix),
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func (r *RunnerImageReconciler) getKanikoPodName(nn types.NamespacedName) string {
	return fmt.Sprintf("jfr-build-%s", nn.Name)
}

func (r *RunnerImageReconciler) getJFRDockerfileConfigMapName(nn types.NamespacedName) string {
	return fmt.Sprintf("jfr-dockerfile-%s", nn.Name)
}

func (r *RunnerImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RunnerImage{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
