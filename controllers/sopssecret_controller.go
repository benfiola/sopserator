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
	"bytes"
	"context"
	sopseratorv1alpha1 "github.com/benfiola/sopserator/api/v1alpha1"
	"github.com/benfiola/sopserator/sops"
	"github.com/benfiola/sopserator/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SOPSSecretReconciler reconciles a SOPSSecret object
type SOPSSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	ownerKey = ".metadata.sopsSecret"
)

// +kubebuilder:rbac:groups=sopserator.benfiola.dev,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sopserator.benfiola.dev,resources=sopssecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:resources=secrets/status,verbs=get;update;patch
func (r *SOPSSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("sopssecret", req.NamespacedName)
	finalizerKey := "sopssecret.finalizers." + sopseratorv1alpha1.GroupVersion.Group

	var sopsSecret sopseratorv1alpha1.SOPSSecret
	if err := r.Client.Get(ctx, req.NamespacedName, &sopsSecret); err != nil {
		log.Error(err, "Failed to retrieve SOPSSecret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle deletion
	if !sopsSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		log = log.WithValues("phase", "cleanup")

		// handle finalizers
		if utils.ListContainsString(sopsSecret.Finalizers, finalizerKey) {
			log.Info("Triggering finalizer")

			// find + delete child secrets
			log.Info("Finding related Secrets")
			var relatedSecrets corev1.SecretList
			if err := r.findRelatedSecrets(ctx, req.Namespace, req.Name, &relatedSecrets); err != nil {
				log.Error(err, "Failed to find related Secrets")
				return ctrl.Result{}, err
			}
			for _, relatedSecret := range relatedSecrets.Items {
				log.Info("Cleaning up Secret")
				if err := r.Delete(ctx, &relatedSecret); err != nil {
					log.Error(err, "Failed to delete related Secret")
					return ctrl.Result{}, err
				}
			}

			// delete finalizer
			utils.ListRemoveString(sopsSecret.Finalizers, finalizerKey)
			sopsSecret.Finalizers = utils.ListRemoveString(sopsSecret.Finalizers, finalizerKey)
			log.Info("Deleting finalizer")
			if err := r.Update(context.Background(), &sopsSecret); err != nil {
				log.Error(err, "Failed to delete finalizer")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// handle creation/update
	log = log.WithValues("phase", "update")

	// add finalizers if missing
	if !utils.ListContainsString(sopsSecret.Finalizers, finalizerKey) {
		log.Info("Adding finalizer")
		sopsSecret.ObjectMeta.Finalizers = append(sopsSecret.ObjectMeta.Finalizers, finalizerKey)
		if err := r.Update(context.Background(), &sopsSecret); err != nil {
			return ctrl.Result{}, err
		}
	}

	// find child secrets
	log.Info("Finding related Secrets")
	var relatedSecrets corev1.SecretList
	if err := r.findRelatedSecrets(ctx, req.Namespace, req.Name, &relatedSecrets); err != nil {
		log.Error(err, "Failed to find related Secrets")
		return ctrl.Result{}, err
	}
	var relatedSecret *corev1.Secret
	exists := len(relatedSecrets.Items) > 0
	if exists {
		log.Info("Related Secret exists")
		relatedSecret = &relatedSecrets.Items[0]
	} else {
		log.Info("Related Secret does not exist")
		relatedSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sopsSecret.GetName(),
				Namespace: sopsSecret.GetNamespace(),
			},
		}
	}

	// decrypt secrets and add them to child secret
	log.Info("Decrypting secrets")
	if err := r.decryptSOPSSecret(&sopsSecret, relatedSecret); err != nil {
		log.Error(err, "Error decrypting secrets")
		return ctrl.Result{}, err
	}

	// ensure secret and sopssecret are related so that changes to one
	// trigger changes in the other.
	log.Info("Setting controller reference")
	if err := ctrl.SetControllerReference(&sopsSecret, relatedSecret, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	// perform create or update child secret
	if exists {
		log.Info("Updating secret")
		if err := r.Update(ctx, relatedSecret); err != nil {
			log.Error(err, "Failed to update Secret")
			return ctrl.Result{}, err
		}
	} else {
		log.Info("Creating secret")
		if err := r.Create(ctx, relatedSecret); err != nil {
			log.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SOPSSecretReconciler) findRelatedSecrets(ctx context.Context, namespace string, sopsSecretName string, secrets *corev1.SecretList) error {
	return r.List(ctx, secrets, client.InNamespace(namespace), client.MatchingFields{ownerKey: sopsSecretName})
}

func (r *SOPSSecretReconciler) decryptSOPSSecret(sopsSecret *sopseratorv1alpha1.SOPSSecret, secret *corev1.Secret) error {
	// marshal sopssecret to YAML string
	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory, nil, r.Scheme,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)
	serializedEncrypted := bytes.NewBuffer([]byte{})
	if err := serializer.Encode(sopsSecret, serializedEncrypted); err != nil {
		return err
	}

	// decrypt YAML string
	serializedDecrypted := bytes.NewBuffer([]byte{})
	if err := sops.DecryptYAML(serializedEncrypted, serializedDecrypted, sops.DecryptYAMLOptions{IgnoreMac: true}); err != nil {
		return err
	}

	// unmarshal decrypted YAML string back into sopssecret
	gvk := sopsSecret.GroupVersionKind()
	decrypted := sopsSecret.DeepCopy()
	var _, _, decodingError = serializer.Decode(serializedDecrypted.Bytes(), &gvk, decrypted)
	if decodingError != nil {
		return decodingError
	}

	// add decrypted data to secret.
	secret.ObjectMeta.Labels = decrypted.Labels
	secret.ObjectMeta.Annotations = decrypted.Annotations
	if decrypted.Data != nil {
		secret.Data = make(map[string][]byte)
		for key := range decrypted.Data {
			secret.Data[key] = bytes.NewBufferString(decrypted.Data[key]).Bytes()
		}
	}
	secret.StringData = decrypted.StringData

	return nil
}

func (r *SOPSSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(&corev1.Secret{}, ownerKey, func(rawObj runtime.Object) []string {
		secret := rawObj.(*corev1.Secret)
		owner := metav1.GetControllerOf(secret)
		if owner == nil {
			return nil
		}
		if owner.Kind != "SOPSSecret" {
			return nil
		}
		return []string{owner.Name}
	})

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&sopseratorv1alpha1.SOPSSecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
