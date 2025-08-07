package s3

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/test/helpers/outputs/adapter/fake"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Generate Vector S3 output", func() {
	var (
		g framework.Generator
	)
	BeforeEach(func() {
		g = framework.MakeGenerator()
	})

	Context("for S3 output", func() {
		It("should generate a basic S3 output with access key authentication", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket: "my-bucket",
						Region: "us-east-1",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeAwsAccessKey,
							AWSAccessKey: &obs.S3AWSAccessKey{
								KeyId: obs.SecretReference{
									Key:        "access_key_id",
									SecretName: "s3-secret",
								},
								KeySecret: obs.SecretReference{
									Key:        "secret_access_key",
									SecretName: "s3-secret",
								},
							},
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-secret": {
					Data: map[string][]byte{
						"access_key_id":     []byte("my-key-id"),
						"secret_access_key": []byte("my-secret-key"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			Expect(conf).To(Not(BeNil()))

			// Generate the configuration
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())
			Expect(results).To(Not(BeEmpty()))

			// Check that it contains S3 sink configuration
			Expect(results).To(ContainSubstring(`type = "aws_s3"`))
			Expect(results).To(ContainSubstring(`bucket = "my-bucket"`))
			Expect(results).To(ContainSubstring(`region = "us-east-1"`))
			Expect(results).To(ContainSubstring(`auth.access_key_id = "SECRET[kubernetes_secret.s3-secret/access_key_id]"`))
			Expect(results).To(ContainSubstring(`auth.secret_access_key = "SECRET[kubernetes_secret.s3-secret/secret_access_key]"`))
		})

		It("should generate S3 output with key prefix", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket:    "my-bucket",
						Region:    "us-west-2",
						KeyPrefix: "logs/",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeAwsAccessKey,
							AWSAccessKey: &obs.S3AWSAccessKey{
								KeyId: obs.SecretReference{
									Key:        "access_key_id",
									SecretName: "s3-secret",
								},
								KeySecret: obs.SecretReference{
									Key:        "secret_access_key",
									SecretName: "s3-secret",
								},
							},
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-secret": {
					Data: map[string][]byte{
						"access_key_id":     []byte("my-key-id"),
						"secret_access_key": []byte("my-secret-key"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())

			Expect(results).To(ContainSubstring(`bucket = "my-bucket"`))
			Expect(results).To(ContainSubstring(`region = "us-west-2"`))
			Expect(results).To(ContainSubstring(`key_prefix = "logs/"`))
		})

		It("should generate S3 output with compression tuning", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket: "my-bucket",
						Region: "us-east-1",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeAwsAccessKey,
							AWSAccessKey: &obs.S3AWSAccessKey{
								KeyId: obs.SecretReference{
									Key:        "access_key_id",
									SecretName: "s3-secret",
								},
								KeySecret: obs.SecretReference{
									Key:        "secret_access_key",
									SecretName: "s3-secret",
								},
							},
						},
						Tuning: &obs.S3TuningSpec{
							Compression: "gzip",
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-secret": {
					Data: map[string][]byte{
						"access_key_id":     []byte("my-key-id"),
						"secret_access_key": []byte("my-secret-key"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())

			Expect(results).To(ContainSubstring(`compression = "gzip"`))
		})

		It("should generate S3 output with IAM role authentication", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket: "my-bucket",
						Region: "us-east-1",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeIAMRole,
							IAMRole: &obs.S3IAMRole{
								RoleARN: obs.SecretReference{
									Key:        "role_arn",
									SecretName: "s3-role-secret",
								},
								Token: obs.BearerToken{
									From: obs.BearerTokenFromServiceAccount,
								},
							},
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-role-secret": {
					Data: map[string][]byte{
						"role_arn": []byte("arn:aws:iam::123456789:role/s3-role"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())

			Expect(results).To(ContainSubstring(`type = "aws_s3"`))
			Expect(results).To(ContainSubstring(`bucket = "my-bucket"`))
			Expect(results).To(ContainSubstring(`region = "us-east-1"`))
		})

		It("should generate S3 output with assume role ARN and service account token", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket: "my-bucket",
						Region: "us-east-1",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeIAMRole,
							IAMRole: &obs.S3IAMRole{
								RoleARN: obs.SecretReference{
									Key:        "role_arn",
									SecretName: "s3-role-secret",
								},
								Token: obs.BearerToken{
									From: obs.BearerTokenFromServiceAccount,
								},
								AssumeRoleARN: &obs.SecretReference{
									Key:        "assume_role_arn",
									SecretName: "assume-role-secret",
								},
							},
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-role-secret": {
					Data: map[string][]byte{
						"role_arn": []byte("arn:aws:iam::123456789:role/s3-role"),
					},
				},
				"assume-role-secret": {
					Data: map[string][]byte{
						"assume_role_arn": []byte("arn:aws:iam::987654321:role/cross-account-role"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())

			Expect(results).To(ContainSubstring(`type = "aws_s3"`))
			Expect(results).To(ContainSubstring(`bucket = "my-bucket"`))
			Expect(results).To(ContainSubstring(`region = "us-east-1"`))
			Expect(results).To(ContainSubstring(`auth.assume_role = "SECRET[kubernetes_secret.assume-role-secret/assume_role_arn]"`))
		})

		It("should generate S3 output with assume role ARN and secret token", func() {
			outputs := []obs.OutputSpec{
				{
					Type: obs.OutputTypeS3,
					Name: "s3-out",
					S3: &obs.S3{
						Bucket: "my-bucket",
						Region: "us-east-1",
						Authentication: &obs.S3Authentication{
							Type: obs.S3AuthTypeIAMRole,
							IAMRole: &obs.S3IAMRole{
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
								AssumeRoleARN: &obs.SecretReference{
									Key:        "assume_role_arn",
									SecretName: "assume-role-secret",
								},
							},
						},
					},
				},
			}
			secrets := map[string]*corev1.Secret{
				"s3-role-secret": {
					Data: map[string][]byte{
						"role_arn": []byte("arn:aws:iam::123456789:role/s3-role"),
					},
				},
				"assume-role-secret": {
					Data: map[string][]byte{
						"assume_role_arn": []byte("arn:aws:iam::987654321:role/cross-account-role"),
					},
				},
				"token-secret": {
					Data: map[string][]byte{
						"token": []byte("my-bearer-token"),
					},
				},
			}
			conf := New("s3out", outputs[0], []string{"application"}, secrets, fake.Output{}, framework.NoOptions)
			results, err := g.GenerateConf(conf...)
			Expect(err).To(BeNil())

			Expect(results).To(ContainSubstring(`type = "aws_s3"`))
			Expect(results).To(ContainSubstring(`bucket = "my-bucket"`))
			Expect(results).To(ContainSubstring(`region = "us-east-1"`))
			Expect(results).To(ContainSubstring(`auth.assume_role = "SECRET[kubernetes_secret.assume-role-secret/assume_role_arn]"`))
		})
	})
})
