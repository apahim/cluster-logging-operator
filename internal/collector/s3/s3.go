package s3

import (
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/collector/common"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/s3"
	corev1 "k8s.io/api/core/v1"
)

// AddAWSEnvironmentVariables adds required AWS environment variables for S3 outputs using IAM role authentication
func AddAWSEnvironmentVariables(collector *corev1.Container, outputs []obs.OutputSpec, secrets observability.Secrets) {
	for _, o := range outputs {
		if o.Type == obs.OutputTypeS3 && o.S3.Authentication != nil && o.S3.Authentication.Type == obs.S3AuthTypeIAMRole {
			if o.S3.Authentication.IAMRole.Token.From == obs.BearerTokenFromServiceAccount {
				if roleARN := s3.ParseRoleArn(o.S3.Authentication, secrets); roleARN != "" {
					tokenPath := common.ServiceAccountBasePath(constants.TokenKey)
					
					// Add AWS environment variables for web identity token authentication
					collector.Env = append(collector.Env,
						corev1.EnvVar{
							Name:  constants.AWSWebIdentityTokenEnvVarKey,
							Value: tokenPath,
						},
						corev1.EnvVar{
							Name:  constants.AWSRoleArnEnvVarKey,
							Value: roleARN,
						},
						corev1.EnvVar{
							Name:  constants.AWSRoleSessionEnvVarKey,
							Value: constants.AWSRoleSessionName,
						},
					)
				}
			}
		}
	}
}