package utils

const (
	Group        = "policy-server.io"
	Version      = "v1alpha2"
	GroupVersion = Group + "/" + Version

	CAFile   = ""
	CertFile = ""
	KeyFile  = ""

	CAEnvVar   = "DB_CA_FILE"
	CertEnvVar = "DB_CERT_FILE"
	KeyEnvVar  = "DB_KEY_FILE"
)

var (
	Endpoints = make([]string, 0)
)
