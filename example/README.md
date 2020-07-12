# Example Workflow

## Pre-requisites

1. Install `gpg` and `sops`
2. Import the [example private key](./example.gpg.key) `gpg --import ./example.gpg.key`
3. Obtain the fingerprint of the key `gpg --list-secret-key`, looking for "Fake SOPSerator Key".  It should be a 40-character alphanumeric string.  Our example key has the fingerprint `9D2EDCE07A39C00F6B84EA7ECF54439B63749658`.

## Create a SOPSKey

1. Export a private key `gpg --export-secret-key --armor <fingerprint>`, copying its output to the clipboard.
2. Paste the output into a SOPSKey resource.  It should look like [this](./key.yaml)

## Create a SOPSSecret

1. Create a Secret resource, writing the file to disk.  Here's an example Secret: [secret.unenc.yaml](./secret.unenc.yaml)
2. Modify the Secret resource.  
   1. `apiVersion: v1` -> `apiVersion: sopserator.benfiola.dev/v1alpha1`
   2. `kind: Secret` -> `kind: SOPSSecret`
3. Encrypt the resource using `sops` + your PGP key fingerprint.  Here's a SOPSSecret created from the above Secret resource: [secret.enc.yaml](./secret.enc.yaml)
```shell script
sops --encrypt \
 --pgp <pgp fingerprint> \
 --encrypted-regex '^(?:data|stringData)$'
 secret.unenc.yaml > secret.enc.yaml
```

## Deployment

1. Create a cluster
2. Deploy CRDs by running `make -C .. install`
3. Deploy the SOPSerator.  `make -C .. run`
4. Apply the SOPSKey resource. `kubectl apply -f key.yaml`.  Ensure `kubectl get sopskeys/secretkey` returns a successfully created resource.
5. Apply the SOPSSecret resource. `kubectl apply -f secret.enc.yaml`.  Ensure `kubectl get sopssecrets/secret` returns a successfully created resource.
6. Ensure `kubectl get secrets/secret` exists, and its data matches that of [secret.unenc.yaml](./secret.unenc.yaml)
