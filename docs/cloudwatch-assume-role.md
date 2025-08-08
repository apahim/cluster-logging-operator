# CloudWatch AssumeRole Configuration

This document describes how to configure CloudWatch output with AssumeRole ARN for cross-account access or role chaining scenarios.

## Overview

The `AssumeRoleARN` field in CloudWatch authentication allows Vector to assume an additional role after authenticating with the primary role. This is useful for:

- Cross-account log forwarding
- Role chaining scenarios
- Maintaining ServiceAccount AWS_ROLE_ARN environment variable behavior while adding explicit assume_role configuration

## Configuration

### ClusterLogForwarder Example

```yaml
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: sample-forwarder
  namespace: openshift-logging
spec:
  outputs:
  - name: cloudwatch-cross-account
    type: cloudwatch
    cloudwatch:
      region: us-east-1
      groupName: my-log-group
      authentication:
        type: iamRole
        iamRole:
          roleARN:
            secretName: primary-role-secret
            key: role_arn
          token:
            from: serviceAccount
          assumeRoleARN:
            secretName: assume-role-secret
            key: assume_role_arn
  pipelines:
  - name: application-logs
    inputRefs:
    - application
    outputRefs:
    - cloudwatch-cross-account
```

### Required Secrets

1. **Primary Role Secret** (`primary-role-secret`):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: primary-role-secret
     namespace: openshift-logging
   data:
     role_arn: <base64-encoded-primary-role-arn>
   ```

2. **Assume Role Secret** (`assume-role-secret`):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: assume-role-secret
     namespace: openshift-logging
   data:
     assume_role_arn: <base64-encoded-cross-account-role-arn>
   ```

### Example Role ARNs

- Primary Role ARN: `arn:aws:iam::111111111111:role/logging-primary-role`
- Assume Role ARN: `arn:aws:iam::222222222222:role/cross-account-logging-role`

## How It Works

1. **ServiceAccount Authentication**: The collector pod uses the ServiceAccount's web identity token to authenticate with AWS
2. **Primary Role**: Vector assumes the primary role specified in `roleARN`
3. **Additional Role**: Vector then assumes the role specified in `assumeRoleARN` using the credentials from the primary role
4. **Log Forwarding**: Logs are forwarded to CloudWatch using the final assumed role's permissions

## Vector Configuration Generated

The above ClusterLogForwarder generates the following Vector configuration:

```toml
[sinks.cloudwatch-cross-account]
type = "aws_cloudwatch_logs"
inputs = ["pipeline_name"]
region = "us-east-1"
group_name = "my-log-group"
stream_name = "{{ stream_name }}"
auth.credentials_file = "/var/run/ocp-collector/config/my-forwarder-aws-creds/credentials"
auth.profile = "output_cloudwatch-cross-account"
auth.assume_role = "SECRET[kubernetes_secret.assume-role-secret/assume_role_arn]"
healthcheck.enabled = false
```

## AWS IAM Configuration

### Primary Role Trust Policy

The primary role must trust the ServiceAccount's OIDC identity:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::111111111111:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLE"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLE:sub": "system:serviceaccount:openshift-logging:logcollector",
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLE:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

### Primary Role Policy

The primary role needs permission to assume the cross-account role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::222222222222:role/cross-account-logging-role"
    }
  ]
}
```

### Cross-Account Role Trust Policy

The cross-account role must trust the primary role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::111111111111:role/logging-primary-role"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

### Cross-Account Role Policy

The cross-account role needs CloudWatch permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:us-east-1:222222222222:*"
    }
  ]
}
```

## Notes

- The `assumeRoleARN` field is optional and can be omitted if no additional role assumption is needed
- This feature works with both ServiceAccount tokens and secret-based tokens
- The feature preserves existing ServiceAccount AWS_ROLE_ARN environment variable behavior
- When using assumeRoleARN, ensure proper IAM trust relationships are configured between roles