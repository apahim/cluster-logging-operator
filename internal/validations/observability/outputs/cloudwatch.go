package outputs

import (
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/cloudwatch"
)

const (
	RoleARNsOpt       = "roleARNs"
	ErrInvalidRoleARN = "CloudWatch RoleARN is invalid"
)

func ValidateCloudWatchAuth(spec obs.OutputSpec, context internalcontext.ForwarderContext) (results []string) {
	secrets := observability.Secrets(context.Secrets)
	authSpec := spec.Cloudwatch.Authentication

	// Validate role ARN
	if authSpec.Type == obs.CloudwatchAuthTypeIAMRole {
		roleArn := cloudwatch.ParseRoleArn(authSpec, secrets)
		if roleArn == "" {
			results = append(results, ErrInvalidRoleARN)
		}
		
		// Validate optional assume role ARN
		if authSpec.IAMRole.AssumeRoleARN != nil {
			if authSpec.IAMRole.AssumeRoleARN.SecretName == "" {
				results = append(results, "assumeRoleARN requires a valid secret reference")
			}
			if authSpec.IAMRole.AssumeRoleARN.Key == "" {
				results = append(results, "assumeRoleARN secret reference requires a valid key")
			}
		}
	}
	return results
}
