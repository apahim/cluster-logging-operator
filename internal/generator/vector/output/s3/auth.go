package s3

import (
	"fmt"
	"path/filepath"

	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	. "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	vectorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
)

const (
	// AWS_CREDENTIALS_FILE is the environment variable name for the AWS credentials file
	AWS_CREDENTIALS_FILE = "AWS_CREDENTIALS_FILE"
	// AWS_SHARED_CREDENTIALS_FILE is the environment variable name for the AWS shared credentials file
	AWS_SHARED_CREDENTIALS_FILE = "AWS_SHARED_CREDENTIALS_FILE"
	// AWS_PROFILE is the environment variable name for the AWS profile
	AWS_PROFILE = "AWS_PROFILE"
	// AWS_WEB_IDENTITY_TOKEN_FILE is the environment variable name for the web identity token file
	AWS_WEB_IDENTITY_TOKEN_FILE = "AWS_WEB_IDENTITY_TOKEN_FILE"
	// AWS_ROLE_ARN is the environment variable name for the role ARN
	AWS_ROLE_ARN = "AWS_ROLE_ARN"
	// AWS_ROLE_SESSION_NAME is the environment variable name for the role session name
	AWS_ROLE_SESSION_NAME = "AWS_ROLE_SESSION_NAME"
	// EnvAWSRegion is the environment variable name for the AWS region
	EnvAWSRegion = "AWS_REGION"
	// CredentialsMount is the mount path for AWS credentials
	CredentialsMount = "/var/run/ocp-collector/aws"
	// CredentialsFile is the filename for AWS credentials
	CredentialsFile = "credentials"
)

type Auth struct {
	ComponentID     string
	KeyID           OptionalPair
	KeySecret       OptionalPair
	CredentialsPath OptionalPair
	Profile         OptionalPair
	AssumeRole      OptionalPair
	auth            obs.S3AuthType
}

func NewAuth() Auth {
	return Auth{
		KeyID:           NewOptionalPair("access_key_id", nil),
		KeySecret:       NewOptionalPair("secret_access_key", nil),
		CredentialsPath: NewOptionalPair("credentials_file", nil),
		Profile:         NewOptionalPair("profile", nil),
		AssumeRole:      NewOptionalPair("assume_role", nil),
	}
}

func (a Auth) Name() string {
	return "awsAuthTemplate"
}

func (a Auth) Template() string {
	if a.auth == obs.S3AuthTypeAwsAccessKey {
		return `{{define "` + a.Name() + `" -}}
[sinks.{{.ComponentID}}.auth]
{{.KeyID}}
{{.KeySecret}}
{{- end}}`
	}
	return `{{define "` + a.Name() + `" -}}
[sinks.{{.ComponentID}}.auth]
{{.KeyID}}
{{.KeySecret}}
{{.CredentialsPath}}
{{.Profile}}
{{.AssumeRole}}
{{- end}}`
}

func AuthConfig(id string, auth *obs.S3Authentication, secrets observability.Secrets) Element {
	a := NewAuth()
	a.ComponentID = id
	switch auth.Type {
	case obs.S3AuthTypeAwsAccessKey:
		a.KeyID = NewOptionalPair("access_key_id", vectorhelpers.SecretFrom(&auth.AWSAccessKey.KeyId))
		a.KeySecret = NewOptionalPair("secret_access_key", vectorhelpers.SecretFrom(&auth.AWSAccessKey.KeySecret))
		a.auth = auth.Type
	case obs.S3AuthTypeIAMRole:
		a.auth = auth.Type
		// IAM Role authentication will be handled by environment variables and credentials files
		if auth.IAMRole.AssumeRoleARN != nil {
			a.AssumeRole = NewOptionalPair("assume_role", vectorhelpers.SecretFrom(auth.IAMRole.AssumeRoleARN))
		}
	}
	return a
}

// GenerateS3CredentialFiles generates the necessary environment variables and file paths for S3 IAM role authentication
func GenerateS3CredentialFiles(id string, auth *obs.S3Authentication, secrets observability.Secrets) map[string]string {
	envVars := make(map[string]string)

	if auth.Type == obs.S3AuthTypeIAMRole && auth.IAMRole != nil {
		// Set up AWS credential file path
		credentialsFilePath := filepath.Join(CredentialsMount, id, CredentialsFile)
		envVars[AWS_CREDENTIALS_FILE] = credentialsFilePath
		envVars[AWS_SHARED_CREDENTIALS_FILE] = credentialsFilePath
		envVars[AWS_PROFILE] = id

		// Set up web identity token if using service account
		if auth.IAMRole.Token.From == obs.BearerTokenFromServiceAccount {
			tokenFile := "/var/run/secrets/kubernetes.io/serviceaccount/token"
			envVars[AWS_WEB_IDENTITY_TOKEN_FILE] = tokenFile
		}

		// Set up role ARN
		// The actual role ARN will be read from the secret at runtime
		envVars[AWS_ROLE_SESSION_NAME] = fmt.Sprintf("cluster-logging-%s", id)
	}

	return envVars
}
