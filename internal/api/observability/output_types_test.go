package observability_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obsv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	. "github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/test"
	"strings"
)

var _ = Describe("helpers for output types", func() {

	Context("#SecretReferences", func() {

		It("should return an empty set of keys when authentication is not defined for an output", func() {
			for _, t := range obsv1.OutputTypes {

				outputType := strings.TrimPrefix("OutputType", string(t))
				outputType = strings.ToLower(outputType[0:1]) + outputType[1:]
				yaml := test.JSONLine(map[string]interface{}{
					"type":     t,
					outputType: map[string]interface{}{},
				})
				spec := &obsv1.OutputSpec{}
				test.MustUnmarshal(yaml, spec)
				Expect(SecretReferences(*spec)).To(BeEmpty())
			}
		})

		Context("for S3 output", func() {
			It("should include AssumeRoleARN secret reference when specified", func() {
				spec := obsv1.OutputSpec{
					Type: obsv1.OutputTypeS3,
					S3: &obsv1.S3{
						Bucket: "test-bucket",
						Region: "us-east-1",
						Authentication: &obsv1.S3Authentication{
							Type: obsv1.S3AuthTypeIAMRole,
							IAMRole: &obsv1.S3IAMRole{
								RoleARN: obsv1.SecretReference{
									SecretName: "primary-role-secret",
									Key:        "role_arn",
								},
								Token: obsv1.BearerToken{
									From: obsv1.BearerTokenFromServiceAccount,
								},
								AssumeRoleARN: &obsv1.SecretReference{
									SecretName: "assume-role-secret",
									Key:        "assume_role_arn",
								},
							},
						},
					},
				}

				secrets := SecretReferences(spec)
				Expect(secrets).To(HaveLen(2))
				
				// Should include primary role ARN
				Expect(secrets).To(ContainElement(&obsv1.SecretReference{
					SecretName: "primary-role-secret",
					Key:        "role_arn",
				}))
				
				// Should include assume role ARN
				Expect(secrets).To(ContainElement(&obsv1.SecretReference{
					SecretName: "assume-role-secret",
					Key:        "assume_role_arn",
				}))
			})

			It("should work without AssumeRoleARN for backward compatibility", func() {
				spec := obsv1.OutputSpec{
					Type: obsv1.OutputTypeS3,
					S3: &obsv1.S3{
						Bucket: "test-bucket",
						Region: "us-east-1",
						Authentication: &obsv1.S3Authentication{
							Type: obsv1.S3AuthTypeIAMRole,
							IAMRole: &obsv1.S3IAMRole{
								RoleARN: obsv1.SecretReference{
									SecretName: "primary-role-secret",
									Key:        "role_arn",
								},
								Token: obsv1.BearerToken{
									From: obsv1.BearerTokenFromServiceAccount,
								},
								// AssumeRoleARN is not specified
							},
						},
					},
				}

				secrets := SecretReferences(spec)
				Expect(secrets).To(HaveLen(1))
				Expect(secrets).To(ContainElement(&obsv1.SecretReference{
					SecretName: "primary-role-secret",
					Key:        "role_arn",
				}))
			})
		})

	})
})
