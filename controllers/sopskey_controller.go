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
	"github.com/benfiola/sopserator/gpg"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sopseratorv1alpha1 "github.com/benfiola/sopserator/api/v1alpha1"
	"github.com/benfiola/sopserator/utils"
)

// SOPSKeyReconciler reconciles a SOPSKey object
type SOPSKeyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sopserator.benfiola.dev,resources=sopskeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sopserator.benfiola.dev,resources=sopskeys/status,verbs=get;update;patch
func (r *SOPSKeyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("sopskey", req.NamespacedName)
	finalizerKey := "sopskey.finalizers." + sopseratorv1alpha1.GroupVersion.Group

	var sopsKey sopseratorv1alpha1.SOPSKey
	if err := r.Get(ctx, req.NamespacedName, &sopsKey); err != nil {
		log.Error(err, "Failed to retrieve SOPSKey")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle delete
	if !sopsKey.ObjectMeta.DeletionTimestamp.IsZero() {
		log = log.WithValues("phase", "cleanup")

		// handle finalizer
		if utils.ListContainsString(sopsKey.Finalizers, finalizerKey) {
			log.Info("Triggering finalizer")

			if (sopsKey.Spec.PGP != sopseratorv1alpha1.SOPSKeySpecPGP{}) {
				log = log.WithValues("keytype", "pgp")
				var fingerprint string
				var err error
				if fingerprint, err = gpg.Fingerprint(bytes.NewBufferString(sopsKey.Spec.PGP.Key)); err != nil {
					log.Error(err, "Unable to obtain fingerprint from key")
					return ctrl.Result{}, err
				}
				if err = gpg.DeleteSecretKey(fingerprint); err != nil {
					log.Error(err, "Unable to delete PGP key")
					return ctrl.Result{}, err
				}
			}

			// remove finalizer
			sopsKey.Finalizers = utils.ListRemoveString(sopsKey.Finalizers, finalizerKey)
			if err := r.Update(ctx, &sopsKey); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// handle create/update
	log = log.WithValues("phase", "update")

	// add finalizers if missing
	if !utils.ListContainsString(sopsKey.Finalizers, finalizerKey) {
		log.Info("Adding finalizer")
		sopsKey.ObjectMeta.Finalizers = append(sopsKey.ObjectMeta.Finalizers, finalizerKey)
		if err := r.Update(ctx, &sopsKey); err != nil {
			return ctrl.Result{}, err
		}
	}

	if (sopsKey.Spec.PGP != sopseratorv1alpha1.SOPSKeySpecPGP{}) {
		log = log.WithValues("keytype", "pgp")

		// import PGP key
		key := bytes.NewBufferString(sopsKey.Spec.PGP.Key)
		if err := gpg.ImportSecretKey(key); err != nil {
			log.Error(err, "Failed to import PGP key")
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *SOPSKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sopseratorv1alpha1.SOPSKey{}).
		Complete(r)
}
