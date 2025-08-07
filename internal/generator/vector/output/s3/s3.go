package s3

import (
	_ "embed"
	"regexp"

	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"
	"github.com/openshift/cluster-logging-operator/internal/utils"

	genhelper "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	. "github.com/openshift/cluster-logging-operator/internal/generator/vector/elements"
	vectorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
)

type S3 struct {
	Desc           string
	ComponentID    string
	Inputs         string
	Region         string
	Bucket         string
	KeyPrefix      string
	SecurityConfig Element
	common.RootMixin
}

func (s S3) Name() string {
	return "s3Template"
}

func (s S3) Template() string {
	return `{{define "` + s.Name() + `" -}}
{{.Desc}}
[sinks.{{.ComponentID}}]
type = "aws_s3"
inputs = {{.Inputs}}
bucket = "{{.Bucket}}"
region = "{{.Region}}"{{if .KeyPrefix}}
key_prefix = "{{.KeyPrefix}}"{{end}}
{{compose_one .SecurityConfig}}
{{.Compression}}
healthcheck.enabled = false
{{end}}`
}

func (s *S3) SetCompression(algo string) {
	s.Compression.Value = algo
}

func New(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, strategy common.ConfigStrategy, op Options) []Element {
	componentID := id
	if genhelper.IsDebugOutput(op) {
		return []Element{
			Debug(id, vectorhelpers.MakeInputs([]string{componentID}...)),
		}
	}
	return []Element{
		NewS3(componentID, o, inputs, secrets, op),
		common.NewEncoding(id, common.CodecJSON),
		common.NewBatch(id, strategy),
	}
}

func NewS3(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, op Options) *S3 {
	return sink(id, o, inputs, secrets, op, o.S3.Region, o.S3.Bucket, o.S3.KeyPrefix)
}

func sink(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, op Options, region, bucket, keyPrefix string) *S3 {
	s := &S3{
		ComponentID:    id,
		Inputs:         vectorhelpers.MakeInputs(inputs...),
		Bucket:         bucket,
		Region:         region,
		KeyPrefix:      keyPrefix,
		SecurityConfig: authConfig(o.Name, o.S3.Authentication, secrets, op),
		RootMixin:      common.NewRootMixin(nil),
	}
	if o.S3.Tuning != nil {
		if o.S3.Tuning.Compression != "" {
			s.SetCompression(o.S3.Tuning.Compression)
		}
	}
	return s
}

func authConfig(outputName string, auth *obs.S3Authentication, secrets observability.Secrets, options Options) Element {
	if auth == nil {
		return Nil
	}

	a := NewAuth()
	switch auth.Type {
	case obs.S3AuthTypeAwsAccessKey:
		a.KeyID = genhelper.NewOptionalPair("auth.access_key_id", vectorhelpers.SecretFrom(&auth.AWSAccessKey.KeyId))
		a.KeySecret = genhelper.NewOptionalPair("auth.secret_access_key", vectorhelpers.SecretFrom(&auth.AWSAccessKey.KeySecret))

	case obs.S3AuthTypeIAMRole:
		switch auth.IAMRole.Token.From {
		case obs.BearerTokenFromServiceAccount:
			// Vector automatically detects web identity token credentials from the environment
			// when running in STS-enabled clusters. The environment variables (AWS_WEB_IDENTITY_TOKEN_FILE,
			// AWS_ROLE_ARN) set by the collector will handle the role assumption automatically.
			// No additional auth configuration needed in Vector config.
		case obs.BearerTokenFromSecret:
			// When using a token from secret, we'll use the credentials file approach
			if forwarderName, found := utils.GetOption(options, OptionForwarderName, ""); found {
				a.CredentialsPath = genhelper.NewOptionalPair("auth.credentials_file", vectorhelpers.ConfigPath(forwarderName+"-"+constants.AWSCredentialsConfigMapName, constants.AWSCredentialsKey))
				a.Profile = genhelper.NewOptionalPair("auth.profile", "output_"+outputName)
			}
		}
	}
	return a
}

// ParseRoleArn search for matching valid ARN
func ParseRoleArn(auth *obs.S3Authentication, secrets observability.Secrets) string {
	if auth.Type == obs.S3AuthTypeIAMRole {
		roleArnString := secrets.AsString(&auth.IAMRole.RoleARN)

		if roleArnString != "" {
			reg := regexp.MustCompile(`(arn:aws(.*)?:(iam|sts)::\d{12}:role\/\S+)\s?`)
			roleArn := reg.FindStringSubmatch(roleArnString)
			if roleArn != nil {
				return roleArn[1] // the capturing group is index 1
			}
		}
	}
	return ""
}
