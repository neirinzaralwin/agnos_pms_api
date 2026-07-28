// Package openapi embeds the OpenAPI 3 contract for the HTTP API.
// The YAML file is the single source of truth for Swagger UI and ReDoc.
package openapi

import _ "embed"

// Spec is the raw OpenAPI document served at /openapi/openapi.yaml.
//
//go:embed openapi.yaml
var Spec []byte
