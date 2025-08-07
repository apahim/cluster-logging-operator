# Troubleshooting S3 Assume Role Configuration

## Problem

Vector crashes with error:
```
Configuration error. error=Error while retrieving secret from backend "kubernetes_secret": No such file or directory (os error 2).
```

## Root Cause

This error occurs when Vector tries to read a secret that is referenced in the configuration but not properly mounted to the collector pod.

## Solutions

### 1. Verify Secret Exists

Ensure the secret referenced in `assumeRoleARN` exists in the correct namespace:

```bash
kubectl get secret <assume-role-secret-name> -n openshift-logging
```

### 2. Check Secret Content

Verify the secret contains the expected key:

```bash
kubectl get secret <assume-role-secret-name> -n openshift-logging -o yaml
```

The secret should contain the key specified in the `assumeRoleARN.key` field.

### 3. Example Working Configuration

```yaml
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: s3-with-assume-role
  namespace: openshift-logging
spec:
  outputs:
  - name: s3-cross-account
    type: s3
    s3:
      bucket: my-cross-account-bucket
      region: us-east-1
      authentication:
        type: iamRole
        iamRole:
          roleARN:
            secretName: s3-role-secret
            key: role_arn
          token:
            from: serviceAccount
          assumeRoleARN:
            secretName: assume-role-secret  # This secret MUST exist
            key: assume_role_arn           # This key MUST exist in the secret
  pipelines:
  - name: forward-to-s3
    inputRefs: [application]
    outputRefs: [s3-cross-account]
---
apiVersion: v1
kind: Secret
metadata:
  name: s3-role-secret
  namespace: openshift-logging
type: Opaque
data:
  role_arn: <base64-encoded-primary-role-arn>
---
apiVersion: v1
kind: Secret
metadata:
  name: assume-role-secret
  namespace: openshift-logging
type: Opaque
data:
  assume_role_arn: <base64-encoded-assume-role-arn>
```

### 4. Generated Vector Configuration

This should generate Vector configuration like:

```toml
[secret.kubernetes_secret]
type = "directory"
path = "/var/run/ocp-collector/secrets"

[sinks.s3_cross_account]
type = "aws_s3"
bucket = "my-cross-account-bucket"
region = "us-east-1"
auth.assume_role = "SECRET[kubernetes_secret.assume-role-secret/assume_role_arn]"
```

### 5. How It Works

- **ServiceAccount Role**: The primary authentication uses the ServiceAccount's `AWS_ROLE_ARN` environment variable
- **Assume Role**: Vector then assumes the additional role specified in `assumeRoleARN`
- **Secret Mounting**: The cluster-logging-operator should mount the secret to `/var/run/ocp-collector/secrets/assume-role-secret/assume_role_arn`

### 6. Debugging Steps

1. **Check if secret is mounted**:
   ```bash
   kubectl exec -n openshift-logging <collector-pod> -- ls -la /var/run/ocp-collector/secrets/
   ```

2. **Check secret content in pod**:
   ```bash
   kubectl exec -n openshift-logging <collector-pod> -- cat /var/run/ocp-collector/secrets/assume-role-secret/assume_role_arn
   ```

3. **Check Vector configuration**:
   ```bash
   kubectl exec -n openshift-logging <collector-pod> -- cat /etc/vector/vector.yaml
   ```

If the secret directory is empty or the file doesn't exist, the issue is in the secret mounting process.