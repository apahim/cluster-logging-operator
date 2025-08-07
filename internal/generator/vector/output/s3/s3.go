package s3

import (
	_ "embed"
	"strings"

	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/utils"

	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common/tls"

	genhelper "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	. "github.com/openshift/cluster-logging-operator/internal/generator/vector/elements"
	vectorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	commontemplate "github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common/template"
)

type Endpoint struct {
	URL string
}

func (e Endpoint) Name() string {
	return "s3EndpointTemplate"
}

func (e Endpoint) Template() string {
	ret := `{{define "` + e.Name() + `" -}}`
	if e.URL != "" {
		ret += `endpoint = "{{ .URL }}"`
	}
	ret += `{{end}}`
	return ret
}

type S3 struct {
	Desc           string
	ComponentID    string
	Inputs         string
	Region         string
	Bucket         string
	KeyPrefix      string
	EndpointConfig Element
	SecurityConfig Element
	common.RootMixin
}

func (s S3) Name() string {
	return "s3Template"
}

func (s S3) Template() string {
	return `{{define "` + s.Name() + `" -}}
{{if .Desc -}}
# {{.Desc}}
{{end -}}
[sinks.{{.ComponentID}}]
type = "aws_s3"
inputs = {{.Inputs}}
region = "{{.Region}}"
bucket = "{{.Bucket}}"
{{if .KeyPrefix -}}
key_prefix = "{{.KeyPrefix}}"
{{end -}}
healthcheck.enabled = false
{{compose_one .EndpointConfig}}
{{.Compression}}
{{compose_one .SecurityConfig}}
{{- end}}`
}

func (s *S3) SetCompression(algo string) {
	s.Compression.Value = algo
}

func New(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, strategy common.ConfigStrategy, op Options) []Element {
	componentID := vectorhelpers.MakeID(id, "normalize_keys")
	keyPrefixID := vectorhelpers.MakeID(id, "key_prefix")
	if genhelper.IsDebugOutput(op) {
		return []Element{
			NormalizeKeys(componentID, inputs, o.S3.KeyPrefix),
			Debug(id, vectorhelpers.MakeInputs([]string{componentID}...)),
		}
	}

	s3Sink := sink(id, o, []string{keyPrefixID}, secrets, op, o.S3.Region, o.S3.Bucket, keyPrefixID)
	if strategy != nil {
		strategy.VisitSink(s3Sink)
	}

	var elements []Element

	// If KeyPrefix is empty or doesn't contain templating, use direct value
	if o.S3.KeyPrefix == "" || !strings.Contains(o.S3.KeyPrefix, "{") {
		// For static key prefixes, use the sink directly with the inputs
		sinkElement := sink(id, o, inputs, secrets, op, o.S3.Region, o.S3.Bucket, o.S3.KeyPrefix)
		if strategy != nil {
			strategy.VisitSink(sinkElement)
		}
		elements = []Element{sinkElement}
	} else {
		// Create the key prefix VRL transform for dynamic key prefixes
		elements = []Element{
			NormalizeKeys(componentID, inputs, o.S3.KeyPrefix),
			commontemplate.TemplateRemap(keyPrefixID, []string{componentID}, o.S3.KeyPrefix, keyPrefixID, "S3 KeyPrefix"),
			s3Sink,
		}
	}

	elements = append(elements,
		common.NewEncoding(id, common.CodecJSON),
		common.NewAcknowledgments(id, strategy),
		common.NewBatch(id, strategy),
		common.NewBuffer(id, strategy),
		common.NewRequest(id, strategy),
		tls.New(id, o.TLS, secrets, op),
	)

	return elements
}

func NewS3(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, op Options) *S3 {
	return sink(id, o, inputs, secrets, op, o.S3.Region, o.S3.Bucket, o.S3.KeyPrefix)
}

func sink(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, op Options, region, bucket, keyPrefixID string) *S3 {
	var compression interface{}
	if o.S3.Tuning != nil && o.S3.Tuning.Compression != "" {
		compression = o.S3.Tuning.Compression
	}

	s3Elem := &S3{
		Desc:           "Sending logs to AWS S3",
		ComponentID:    id,
		Inputs:         vectorhelpers.MakeInputs(inputs...),
		Region:         region,
		Bucket:         bucket,
		KeyPrefix:      keyPrefixID,
		EndpointConfig: Endpoint{URL: o.S3.URL},
		RootMixin:      common.NewRootMixin(compression),
	}

	// Set authentication config
	if o.S3 != nil && o.S3.Authentication != nil {
		s3Elem.SecurityConfig = AuthConfig(id, o.S3.Authentication, secrets)
	} else {
		s3Elem.SecurityConfig = Nil
	}

	return s3Elem
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
			// If an explicit assume_role ARN is provided, it will be used in addition to the ServiceAccount role.
			if auth.IAMRole.AssumeRoleARN != nil {
				a.AssumeRole = genhelper.NewOptionalPair("auth.assume_role", vectorhelpers.SecretFrom(auth.IAMRole.AssumeRoleARN))
			}
		case obs.BearerTokenFromSecret:
			// When using a token from secret, we'll use the credentials file approach
			if forwarderName, found := utils.GetOption(options, OptionForwarderName, ""); found {
				a.CredentialsPath = genhelper.NewOptionalPair("auth.credentials_file", vectorhelpers.ConfigPath(forwarderName+"-"+constants.AWSCredentialsConfigMapName, constants.AWSCredentialsKey))
				a.Profile = genhelper.NewOptionalPair("auth.profile", "output_"+outputName)
			}
			// Also support assume_role with credential file approach
			if auth.IAMRole.AssumeRoleARN != nil {
				a.AssumeRole = genhelper.NewOptionalPair("auth.assume_role", vectorhelpers.SecretFrom(auth.IAMRole.AssumeRoleARN))
			}
		}
	}
	return a
}

// NormalizeKeys generates a VRL script to create the key structure based on log type and fields
func NormalizeKeys(id string, inputs []string, keyPrefix string) Element {
	return &VectorScript{
		ComponentID: id,
		Inputs:      vectorhelpers.MakeInputs(inputs...),
		Script: `
# Create S3 object key structure
if !exists(.file) { .file = "logs" }

# Set up date components for key prefix
._internal.year = format_timestamp!(.timestamp, format: "%Y")
._internal.month = format_timestamp!(.timestamp, format: "%m") 
._internal.day = format_timestamp!(.timestamp, format: "%d")

# Default key prefix pattern
._internal.key_prefix = join!(["date=", ._internal.year, "-", ._internal.month, "-", ._internal.day, "/"])
`,
	}
}

// VectorScript is a VRL remap transform for Vector
type VectorScript struct {
	ComponentID string
	Inputs      string
	Script      string
}

func (t VectorScript) Name() string {
	return "vectorScriptTemplate"
}

func (t VectorScript) Template() string {
	return `{{define "` + t.Name() + `" -}}
[transforms.{{.ComponentID}}]
type = "remap"
inputs = {{.Inputs}}
source = '''{{.Script}}'''
{{- end}}
`
}
