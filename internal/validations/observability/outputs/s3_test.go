package outputs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
)

var _ = Describe("ValidateS3Auth", func() {
	var (
		output  obs.OutputSpec
		context internalcontext.ForwarderContext
	)

	BeforeEach(func() {
		output = obs.OutputSpec{
			Type: obs.OutputTypeS3,
			Name: "test-s3",
		}
		context = internalcontext.ForwarderContext{}
	})

	Context("when S3 configuration is missing", func() {
		It("should return validation error", func() {
			output.S3 = nil
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("s3 authentication is required"))
		})

		It("should return error when authentication is missing", func() {
			output.S3 = &obs.S3{
				Bucket: "test-bucket",
				Region: "us-east-1",
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("s3 authentication is required"))
		})
	})

	Context("when using AWS access key authentication", func() {
		BeforeEach(func() {
			output.S3 = &obs.S3{
				Bucket: "test-bucket",
				Region: "us-east-1",
				Authentication: &obs.S3Authentication{
					Type: obs.S3AuthTypeAwsAccessKey,
				},
			}
		})

		It("should require awsAccessKey config", func() {
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("awsAccessKey authentication method requires awsAccessKey config"))
		})

		It("should require keyId secret reference", func() {
			output.S3.Authentication.AWSAccessKey = &obs.S3AWSAccessKey{
				KeySecret: obs.SecretReference{
					Key:        "secret_access_key",
					SecretName: "s3-secret",
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("awsAccessKey authentication requires keyId secret reference"))
		})

		It("should require keySecret secret reference", func() {
			output.S3.Authentication.AWSAccessKey = &obs.S3AWSAccessKey{
				KeyId: obs.SecretReference{
					Key:        "access_key_id",
					SecretName: "s3-secret",
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("awsAccessKey authentication requires keySecret secret reference"))
		})

		It("should pass validation when both secrets are provided", func() {
			output.S3.Authentication.AWSAccessKey = &obs.S3AWSAccessKey{
				KeyId: obs.SecretReference{
					Key:        "access_key_id",
					SecretName: "s3-secret",
				},
				KeySecret: obs.SecretReference{
					Key:        "secret_access_key",
					SecretName: "s3-secret",
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(BeEmpty())
		})
	})

	Context("when using IAM role authentication", func() {
		BeforeEach(func() {
			output.S3 = &obs.S3{
				Bucket: "test-bucket",
				Region: "us-east-1",
				Authentication: &obs.S3Authentication{
					Type: obs.S3AuthTypeIAMRole,
				},
			}
		})

		It("should require iamRole config", func() {
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("iamRole authentication method requires iamRole config"))
		})

		It("should require roleARN secret reference", func() {
			output.S3.Authentication.IAMRole = &obs.S3IAMRole{
				Token: obs.BearerToken{
					From: obs.BearerTokenFromServiceAccount,
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("iamRole authentication requires roleARN secret reference"))
		})

		It("should require secret config when token is from secret", func() {
			output.S3.Authentication.IAMRole = &obs.S3IAMRole{
				RoleARN: obs.SecretReference{
					Key:        "role_arn",
					SecretName: "s3-role-secret",
				},
				Token: obs.BearerToken{
					From: obs.BearerTokenFromSecret,
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("iamRole authentication with token from secret requires secret config"))
		})

		It("should pass validation with service account token", func() {
			output.S3.Authentication.IAMRole = &obs.S3IAMRole{
				RoleARN: obs.SecretReference{
					Key:        "role_arn",
					SecretName: "s3-role-secret",
				},
				Token: obs.BearerToken{
					From: obs.BearerTokenFromServiceAccount,
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(BeEmpty())
		})

		It("should pass validation with secret token", func() {
			output.S3.Authentication.IAMRole = &obs.S3IAMRole{
				RoleARN: obs.SecretReference{
					Key:        "role_arn",
					SecretName: "s3-role-secret",
				},
				Token: obs.BearerToken{
					From: obs.BearerTokenFromSecret,
					Secret: &obs.BearerTokenSecretKey{
						Key:  "token",
						Name: "token-secret",
					},
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(BeEmpty())
		})
	})

	Context("when using invalid authentication type", func() {
		It("should return validation error", func() {
			output.S3 = &obs.S3{
				Bucket: "test-bucket",
				Region: "us-east-1",
				Authentication: &obs.S3Authentication{
					Type: "invalid",
				},
			}
			messages := ValidateS3Auth(output, context)
			Expect(messages).To(ContainElement("s3 authentication type must be 'awsAccessKey' or 'iamRole'"))
		})
	})
})