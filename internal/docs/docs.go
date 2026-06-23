package docs

import _ "embed"

// OpenAPIJSON is the API description consumed by the Scalar UI.
//
//go:embed openapi.json
var OpenAPIJSON []byte
