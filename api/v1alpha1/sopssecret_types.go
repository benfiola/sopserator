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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SOPSSecretData map[string]string

type SOPSSecretStringData map[string]string

type SOPSSecretSOPS struct {
	// +required
	MAC string `json:"mac"`
	// +optional
	EncryptedRegex string `json:"encrypted_regex"`
	// +required
	Version string `json:"version"`
	// +optional
	LastModified string `json:"lastmodified"`
	// +optional
	AzureKV []SOPSSecretSOPSAzureKV `json:"azure_kv"`
	// +optional
	GCPKMS []SOPSSecretSOPSGCPKMS `json:"gcp_kms"`
	// +optional
	KMS []SOPSSecretSOPSKMS `json:"kms"`
	// +optional
	PGP []SOPSSecretSOPSPGP `json:"pgp"`
}

type SOPSSecretSOPSAzureKV struct {
}

type SOPSSecretSOPSGCPKMS struct {
}

type SOPSSecretSOPSKMS struct {
}

type SOPSSecretSOPSPGP struct {
	// +optional
	CreatedAt string `json:"created_at"`
	// +required
	Enc string `json:"enc"`
	// +required
	FP string `json:"fp"`
}

type SOPSSecretStatus struct {
	// +optional
	Secret corev1.ObjectReference   `json:"secret,omitempty"`
	Keys   []corev1.ObjectReference `json:"keys,omitempty"`
}

// +kubebuilder:object:root=true
type SOPSSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data       SOPSSecretData       `json:"data,omitempty"`
	StringData SOPSSecretStringData `json:"stringData,omitempty"`
	Status     SOPSSecretStatus     `json:"status,omitempty"`
	Sops       SOPSSecretSOPS       `json:"sops,omitempty"`
}

// +kubebuilder:object:root=true
type SOPSSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SOPSSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SOPSSecret{}, &SOPSSecretList{})
}
