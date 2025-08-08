package outputs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("validating CloudWatch auth", func() {
	Context("#ValidateCloudWatchAuth", func() {

		var (
			myRoleArn      = "arn:aws:iam::123456789012:role/my-role-to-assume"
			invalidRoleARN = "arn:aws:iam::123456789:role/other-role-to-assume"
			spec           = obs.OutputSpec{
				Name: "output",
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Authentication: &obs.CloudwatchAuthentication{
						Type: obs.CloudwatchAuthTypeIAMRole,
						IAMRole: &obs.CloudwatchIAMRole{
							RoleARN: obs.SecretReference{
								SecretName: "foo",
								Key:        constants.AWSCredentialsKey,
							},
							Token: obs.BearerToken{
								From: obs.BearerTokenFromServiceAccount,
							},
						},
					},
				},
			}

			fooSecret = &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name: "foo",
				},
				Data: map[string][]byte{
					constants.AWSCredentialsKey: []byte(myRoleArn),
					"invalidRoleARN":            []byte(invalidRoleARN),
				},
			}
			context = internalcontext.ForwarderContext{
				Forwarder: &obs.ClusterLogForwarder{
					Spec: obs.ClusterLogForwarderSpec{
						Outputs: []obs.OutputSpec{spec},
					},
				},
				Secrets: map[string]*corev1.Secret{
					fooSecret.Name: fooSecret,
				},
			}
		)

		It("should pass with valid role arn", func() {
			res := ValidateCloudWatchAuth(spec, context)
			Expect(res).To(BeEmpty())
		})

		It("should fail if Role ARN is invalid", func() {
			invalidSpec := obs.OutputSpec{
				Name: "output",
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Authentication: &obs.CloudwatchAuthentication{
						Type: obs.CloudwatchAuthTypeIAMRole,
						IAMRole: &obs.CloudwatchIAMRole{
							RoleARN: obs.SecretReference{
								SecretName: "foo",
								Key:        "invalidRoleARN",
							},
							Token: obs.BearerToken{
								From: obs.BearerTokenFromServiceAccount,
							},
						},
					},
				},
			}
			res := ValidateCloudWatchAuth(invalidSpec, context)
			Expect(res).ToNot(BeEmpty())
			Expect(len(res)).To(BeEquivalentTo(1))
			Expect(res[0]).To(BeEquivalentTo(ErrInvalidRoleARN))
		})

		It("should pass with valid AssumeRoleARN", func() {
			assumeRoleSpec := obs.OutputSpec{
				Name: "output",
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Authentication: &obs.CloudwatchAuthentication{
						Type: obs.CloudwatchAuthTypeIAMRole,
						IAMRole: &obs.CloudwatchIAMRole{
							RoleARN: obs.SecretReference{
								SecretName: "foo",
								Key:        constants.AWSCredentialsKey,
							},
							Token: obs.BearerToken{
								From: obs.BearerTokenFromServiceAccount,
							},
							AssumeRoleARN: &obs.SecretReference{
								SecretName: "assume-role-secret",
								Key:        "assume_role_arn",
							},
						},
					},
				},
			}
			assumeRoleContext := internalcontext.ForwarderContext{
				Secrets: map[string]*corev1.Secret{
					"foo": fooSecret,
					"assume-role-secret": {
						ObjectMeta: v1.ObjectMeta{
							Name: "assume-role-secret",
						},
						Data: map[string][]byte{
							"assume_role_arn": []byte("arn:aws:iam::987654321098:role/cross-account-role"),
						},
					},
				},
			}
			res := ValidateCloudWatchAuth(assumeRoleSpec, assumeRoleContext)
			Expect(res).To(BeEmpty())
		})

		It("should fail if AssumeRoleARN has empty secret name", func() {
			emptySecretSpec := obs.OutputSpec{
				Name: "output",
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Authentication: &obs.CloudwatchAuthentication{
						Type: obs.CloudwatchAuthTypeIAMRole,
						IAMRole: &obs.CloudwatchIAMRole{
							RoleARN: obs.SecretReference{
								SecretName: "foo",
								Key:        constants.AWSCredentialsKey,
							},
							Token: obs.BearerToken{
								From: obs.BearerTokenFromServiceAccount,
							},
							AssumeRoleARN: &obs.SecretReference{
								SecretName: "",
								Key:        "assume_role_arn",
							},
						},
					},
				},
			}
			res := ValidateCloudWatchAuth(emptySecretSpec, context)
			Expect(res).ToNot(BeEmpty())
			Expect(res).To(ContainElement("assumeRoleARN requires a valid secret reference"))
		})

		It("should fail if AssumeRoleARN has empty key", func() {
			emptyKeySpec := obs.OutputSpec{
				Name: "output",
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Authentication: &obs.CloudwatchAuthentication{
						Type: obs.CloudwatchAuthTypeIAMRole,
						IAMRole: &obs.CloudwatchIAMRole{
							RoleARN: obs.SecretReference{
								SecretName: "foo",
								Key:        constants.AWSCredentialsKey,
							},
							Token: obs.BearerToken{
								From: obs.BearerTokenFromServiceAccount,
							},
							AssumeRoleARN: &obs.SecretReference{
								SecretName: "assume-role-secret",
								Key:        "",
							},
						},
					},
				},
			}
			res := ValidateCloudWatchAuth(emptyKeySpec, context)
			Expect(res).ToNot(BeEmpty())
			Expect(res).To(ContainElement("assumeRoleARN secret reference requires a valid key"))
		})

		It("should work without AssumeRoleARN for backward compatibility", func() {
			// AssumeRoleARN is not specified (nil) - use original spec
			res := ValidateCloudWatchAuth(spec, context)
			Expect(res).To(BeEmpty())
		})
	})
})
