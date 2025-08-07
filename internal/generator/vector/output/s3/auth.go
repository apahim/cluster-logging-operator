package s3

import (
	. "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
)

type Auth struct {
	KeyID           OptionalPair
	KeySecret       OptionalPair
	CredentialsPath OptionalPair
	Profile         OptionalPair
	AssumeRole      OptionalPair
}

func NewAuth() Auth {
	return Auth{
		KeyID:           NewOptionalPair("auth.access_key_id", nil),
		KeySecret:       NewOptionalPair("auth.secret_access_key", nil),
		CredentialsPath: NewOptionalPair("auth.credentials_file", nil),
		Profile:         NewOptionalPair("auth.profile", nil),
		AssumeRole:      NewOptionalPair("auth.assume_role", nil),
	}
}

func (a Auth) Name() string {
	return "awsAuthTemplate"
}

func (a Auth) Template() string {
	return `{{define "` + a.Name() + `" -}}
{{.KeyID}}
{{.KeySecret}}
{{.CredentialsPath}}
{{.Profile}}
{{.AssumeRole}}
{{- end}}`
}
