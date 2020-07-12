# SOPSerator

SOPSerator is a Kubernetes operator intended to facilitate [SOPS](https://github.com/mozilla/sops) on a remote Kubernetes cluster.  It does so by providing primitives for submitting encryption keys to the remote cluster. These keys can then be used to decrypt SOPS-encrypted Secret resources. 

This operator was motivated by enabling Secret resources to be written safely to disk.  If they can be written safely to disk, they can then be committed and pushed to remote repositories, where they can be deployed via workflows like [GitOps](https://www.weave.works/technologies/gitops/).  

## Usage

A full example workflow is provided [here](./example/README.md)

## CRDs

To enable this behavior, a few CRDs are needed.

### SOPSKey

A SOPSKey is an encryption key to be used with SOPS.  

#### PGP

```yaml
apiVersion: sopserator.benfiola.dev/v1alpha1
kind: SOPSKey
metadata:
  creationTimestamp: null
  name: secretkey
spec:
  pgp:
    key: |
      # output from gpg --export-secret-key --armor <fingerprint_id>
```

### SOPSSecret

A SOPSSecret is a SOPS-encrypted Secret resource - as a result, do the following to create a SOPSSecret:

1. Create a Secret resource
2. Change:
   1. `kind: Secret` -> `kind: SOPSSecret`
   2. `apiVersion: v1` -> `apiversion: sopserator.benfiola.dev/v1alpha1`
3. Encrypt using `sops --encrypt --encrypted-regex '^(?:data|stringData)$'`

```yaml
apiVersion: sopserator.benfiola.dev/v1alpha1
kind: SOPSSecret
metadata:
    name: secret
data:
    key: ENC[AES256_GCM,data:sjYwxRg=,iv:8NPmLLDB4fTeAhzmOUv5ntYYBImlr2Uw/yXuKfJUoGI=,tag:N+GNXKX5SRsB3DOW5dTl3A==,type:str]
    otherKey: ENC[AES256_GCM,data:OvEaSDXuFqTCtqf5,iv:BOBhs9gfVbDI63nKvCS1Gj4gopseEMlHTVFPe4laQyE=,tag:yWEGah65IBohOv/AXuvUNA==,type:str]
# this metadata is automatically provided by SOPS
sops:
    kms: []
    gcp_kms: []
    azure_kv: []
    lastmodified: ...
    mac: ...
    pgp:
    -   created_at: ...
        enc: |
            ...
        fp: ...
    encrypted_regex: ^(?:data|stringData)$
    version: 3.5.0
```

Once applied to a cluster, a child Secret resource is made containing the corresponding decrypted
 data.  Modifications made to the SOPSSecret resource propagate down to the child Secret resource.

## TODO

1. Add support for Azure KV
2. Add support for GCP KMS
3. Add support for KMS
4. Create CLI simplifying SOPSSecret creation process
5. Tests
