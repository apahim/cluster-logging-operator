package outputs

import (
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
)

// ValidateS3Auth validates S3 authentication configuration
func ValidateS3Auth(output obs.OutputSpec, context internalcontext.ForwarderContext) (messages []string) {
	if output.S3 == nil || output.S3.Authentication == nil {
		return append(messages, "s3 authentication is required")
	}

	auth := output.S3.Authentication
	switch auth.Type {
	case obs.S3AuthTypeAwsAccessKey:
		if auth.AWSAccessKey == nil {
			return append(messages, "awsAccessKey authentication method requires awsAccessKey config")
		}
		// Check that both key ID and secret are provided
		if auth.AWSAccessKey.KeyId.SecretName == "" {
			return append(messages, "awsAccessKey authentication requires keyId secret reference")
		}
		if auth.AWSAccessKey.KeySecret.SecretName == "" {
			return append(messages, "awsAccessKey authentication requires keySecret secret reference")
		}

	case obs.S3AuthTypeIAMRole:
		if auth.IAMRole == nil {
			return append(messages, "iamRole authentication method requires iamRole config")
		}
		// Check that role ARN is provided
		if auth.IAMRole.RoleARN.SecretName == "" {
			return append(messages, "iamRole authentication requires roleARN secret reference")
		}
		// Validate token configuration
		switch auth.IAMRole.Token.From {
		case obs.BearerTokenFromSecret:
			if auth.IAMRole.Token.Secret == nil {
				return append(messages, "iamRole authentication with token from secret requires secret config")
			}
		case obs.BearerTokenFromServiceAccount:
			// Service account tokens are handled by the platform, no additional validation needed
		default:
			return append(messages, "iamRole authentication token must be from 'secret' or 'serviceAccount'")
		}

	default:
		return append(messages, "s3 authentication type must be 'awsAccessKey' or 'iamRole'")
	}

	return messages
}
