/*
JWK functions: jwk_encode, jwk_decode, jwk_thumbprint, jwks.

These wrap go-jose's JSONWebKey (RFC 7517) marshalling and its RFC 7638 canonical thumbprint, which handle the fiddly per-key-type JWK member encoding (EC x/y, RSA n/e, OKP x, and the private members). The Terraform boundary sees ordinary objects and strings; the JWK JSON shape is produced and consumed through go-jose so it stays spec-correct.
*/

package cryptography

import (
	"context"
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"fmt"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// --- jwk_encode -------------------------------------------------------------

var _ function.Function = (*JWKEncodeFunction)(nil)

//go:embed descriptions/jwk_encode.md
var jwkEncodeDescription string

type JWKEncodeFunction struct{}

func NewJWKEncodeFunction() function.Function { return &JWKEncodeFunction{} }

func (f *JWKEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwk_encode"
}

func (f *JWKEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a PEM public or private key (EC, RSA, or Ed25519) to a JWK object",
		MarkdownDescription: jwkEncodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pem", Description: "A PEM-encoded key: `PUBLIC KEY`, `CERTIFICATE`, or a private key (`PRIVATE KEY`, `EC PRIVATE KEY`, `RSA PRIVATE KEY`). EC P-256/384/521, RSA, and Ed25519 are supported."},
		},
		VariadicParameter: function.DynamicParameter{Name: "options", Description: "Optional object setting JWK metadata: `kid`, `use` (for example `sig` or `enc`), and `alg`. Pass at most one."},
		Return:            function.DynamicReturn{},
	}
}

func (f *JWKEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pemStr string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pemStr, &optsArgs))
	if resp.Error != nil {
		return
	}
	if len(optsArgs) > 1 {
		resp.Error = function.NewArgumentFuncError(1, "at most one options object may be provided")
		return
	}

	key, err := parseAnyKeyPEM([]byte(pemStr))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	jwk := jose.JSONWebKey{Key: key}
	if len(optsArgs) == 1 {
		if ferr := applyJWKOptions(&jwk, optsArgs[0].UnderlyingValue()); ferr != nil {
			resp.Error = ferr
			return
		}
	}

	obj, err := joseKeyToTerraform(&jwk)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// --- jwk_decode -------------------------------------------------------------

var _ function.Function = (*JWKDecodeFunction)(nil)

//go:embed descriptions/jwk_decode.md
var jwkDecodeDescription string

type JWKDecodeFunction struct{}

func NewJWKDecodeFunction() function.Function { return &JWKDecodeFunction{} }

func (f *JWKDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwk_decode"
}

func (f *JWKDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a JWK object back to a PEM key (round-trip pair with jwk_encode)",
		MarkdownDescription: jwkDecodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "jwk", Description: "A JWK object (as produced by `jwk_encode`)."},
		},
		VariadicParameter: function.DynamicParameter{Name: "options", Description: "Reserved for future use; no options are currently defined. Pass at most one."},
		Return:            function.StringReturn{},
	}
}

func (f *JWKDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var jwkArg types.Dynamic
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &jwkArg, &optsArgs))
	if resp.Error != nil {
		return
	}
	if hasUnknownValue(jwkArg) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}
	if len(optsArgs) > 1 {
		resp.Error = function.NewArgumentFuncError(1, "at most one options object may be provided")
		return
	}

	jwk, err := joseKeyFromTFObject(jwkArg.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	pemStr, err := joseKeyToPEM(jwk)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &pemStr))
}

// --- jwk_thumbprint ---------------------------------------------------------

var _ function.Function = (*JWKThumbprintFunction)(nil)

//go:embed descriptions/jwk_thumbprint.md
var jwkThumbprintDescription string

type JWKThumbprintFunction struct{}

func NewJWKThumbprintFunction() function.Function { return &JWKThumbprintFunction{} }

func (f *JWKThumbprintFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwk_thumbprint"
}

func (f *JWKThumbprintFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Compute the RFC 7638 canonical JWK thumbprint (base64url); the standard `kid`",
		MarkdownDescription: jwkThumbprintDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "key", Description: "The key: either a PEM string or a JWK object."},
		},
		VariadicParameter: function.StringParameter{Name: "hash", Description: "Hash algorithm: \"SHA-256\" (default), \"SHA-1\", \"SHA-384\", or \"SHA-512\". Pass at most one."},
		Return:            function.StringReturn{},
	}
}

func (f *JWKThumbprintFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var keyArg types.Dynamic
	var hashArgs []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &keyArg, &hashArgs))
	if resp.Error != nil {
		return
	}
	if hasUnknownValue(keyArg) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}
	if len(hashArgs) > 1 {
		resp.Error = function.NewArgumentFuncError(1, "at most one hash may be provided")
		return
	}
	hashName := "SHA-256"
	if len(hashArgs) == 1 {
		hashName = hashArgs[0]
	}
	hashAlg, ok := thumbprintHash(hashName)
	if !ok {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("hash must be one of SHA-256, SHA-1, SHA-384, SHA-512; got %q", hashName))
		return
	}

	jwk, err := joseKeyFromTFKeyArg(keyArg.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	tp, err := jwk.Thumbprint(hashAlg)
	if err != nil {
		resp.Error = function.NewFuncError("thumbprint: " + err.Error())
		return
	}
	out := b64uEncode(tp)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// --- jwks -------------------------------------------------------------------

var _ function.Function = (*JWKSFunction)(nil)

//go:embed descriptions/jwks.md
var jwksDescription string

type JWKSFunction struct{}

func NewJWKSFunction() function.Function { return &JWKSFunction{} }

func (f *JWKSFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwks"
}

func (f *JWKSFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Assemble a JWK Set: { keys = [ ...jwk... ] } from PEM strings or JWK objects",
		MarkdownDescription: jwksDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "keys", Description: "A list of keys, each either a PEM string or a JWK object."},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JWKSFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var keysArg types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &keysArg))
	if resp.Error != nil {
		return
	}
	if hasUnknownValue(keysArg) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicUnknown()))
		return
	}

	elems, err := elementsOf(keysArg.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	jwkVals := make([]attr.Value, 0, len(elems))
	jwkTypes := make([]attr.Type, 0, len(elems))
	for i, e := range elems {
		jwk, err := joseKeyFromTFKeyArg(e)
		if err != nil {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("keys[%d]: %s", i, err.Error()))
			return
		}
		v, err := joseKeyToTerraform(jwk)
		if err != nil {
			resp.Error = function.NewFuncError(fmt.Sprintf("keys[%d]: %s", i, err.Error()))
			return
		}
		jwkVals = append(jwkVals, v)
		jwkTypes = append(jwkTypes, v.Type(nil))
	}

	keysTuple := types.TupleValueMust(jwkTypes, jwkVals)
	obj, diags := types.ObjectValue(
		map[string]attr.Type{"keys": keysTuple.Type(nil)},
		map[string]attr.Value{"keys": keysTuple},
	)
	if diags.HasError() {
		resp.Error = function.NewFuncError("building JWKS object: " + diagsToString(diags))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// --- shared JWK helpers -----------------------------------------------------

// parseAnyKeyPEM parses a PEM block into a Go key value suitable for jose.JSONWebKey.Key: a public key from a `PUBLIC KEY` / `CERTIFICATE`, or a private key from `PRIVATE KEY` / `EC PRIVATE KEY` / `RSA PRIVATE KEY`.
func parseAnyKeyPEM(pemBytes []byte) (interface{}, error) {
	block, rest := pem.Decode(pemBytes)
	for block != nil {
		switch block.Type {
		case "PUBLIC KEY":
			return x509.ParsePKIXPublicKey(block.Bytes)
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse certificate: %w", err)
			}
			return cert.PublicKey, nil
		case "PRIVATE KEY":
			return x509.ParsePKCS8PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			return x509.ParseECPrivateKey(block.Bytes)
		case "RSA PRIVATE KEY":
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		}
		block, rest = pem.Decode(rest)
	}
	return nil, fmt.Errorf("no supported key block found (want PUBLIC KEY, CERTIFICATE, PRIVATE KEY, EC PRIVATE KEY, or RSA PRIVATE KEY)")
}

// joseKeyToTerraform marshals a JSONWebKey to its JWK JSON and converts that to a Terraform object value.
func joseKeyToTerraform(jwk *jose.JSONWebKey) (attr.Value, error) {
	b, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal JWK: %w", err)
	}
	return jsonBytesToTerraform(b)
}

// joseKeyFromTFObject converts a Terraform JWK object into a JSONWebKey via its JSON representation.
func joseKeyFromTFObject(v attr.Value) (*jose.JSONWebKey, error) {
	goVal, err := terraformToGoJSON(v)
	if err != nil {
		return nil, fmt.Errorf("convert JWK object: %w", err)
	}
	if _, ok := goVal.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("jwk must be an object")
	}
	b, err := json.Marshal(goVal)
	if err != nil {
		return nil, fmt.Errorf("encode JWK JSON: %w", err)
	}
	var jwk jose.JSONWebKey
	if err := jwk.UnmarshalJSON(b); err != nil {
		return nil, fmt.Errorf("parse JWK: %w", err)
	}
	return &jwk, nil
}

// joseKeyFromTFKeyArg accepts either a PEM string or a JWK object and returns a JSONWebKey. Used by jwk_thumbprint and jwks where the input may be either form.
func joseKeyFromTFKeyArg(v attr.Value) (*jose.JSONWebKey, error) {
	if dv, ok := v.(basetypes.DynamicValue); ok {
		v = dv.UnderlyingValue()
	}
	if sv, ok := v.(basetypes.StringValue); ok {
		key, err := parseAnyKeyPEM([]byte(sv.ValueString()))
		if err != nil {
			return nil, err
		}
		return &jose.JSONWebKey{Key: key}, nil
	}
	if _, ok := v.(basetypes.ObjectValue); ok {
		return joseKeyFromTFObject(v)
	}
	return nil, fmt.Errorf("key must be a PEM string or a JWK object, got %T", v)
}

// joseKeyToPEM renders a JSONWebKey's underlying key as PEM: a PKIX `PUBLIC KEY` for a public key, a PKCS#8 `PRIVATE KEY` for a private key.
func joseKeyToPEM(jwk *jose.JSONWebKey) (string, error) {
	if jwk.Key == nil {
		return "", fmt.Errorf("jwk has no key material")
	}
	if jwk.IsPublic() {
		der, err := x509.MarshalPKIXPublicKey(jwk.Key)
		if err != nil {
			return "", fmt.Errorf("marshal public key: %w", err)
		}
		return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
	}
	der, err := x509.MarshalPKCS8PrivateKey(jwk.Key)
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

// applyJWKOptions overlays kid / use / alg from an options object onto a JSONWebKey.
func applyJWKOptions(jwk *jose.JSONWebKey, v attr.Value) *function.FuncError {
	if v == nil || v.IsNull() {
		return nil
	}
	obj, ok := v.(basetypes.ObjectValue)
	if !ok {
		return function.NewArgumentFuncError(1, "options must be an object")
	}
	attrs := obj.Attributes()
	if s, present, err := optionalStringAttr(attrs, "kid"); err != nil {
		return function.NewArgumentFuncError(1, err.Error())
	} else if present {
		jwk.KeyID = s
	}
	if s, present, err := optionalStringAttr(attrs, "use"); err != nil {
		return function.NewArgumentFuncError(1, err.Error())
	} else if present {
		jwk.Use = s
	}
	if s, present, err := optionalStringAttr(attrs, "alg"); err != nil {
		return function.NewArgumentFuncError(1, err.Error())
	} else if present {
		jwk.Algorithm = s
	}
	return nil
}

// elementsOf returns the elements of a list or tuple attr.Value.
func elementsOf(v attr.Value) ([]attr.Value, error) {
	if dv, ok := v.(basetypes.DynamicValue); ok {
		v = dv.UnderlyingValue()
	}
	switch val := v.(type) {
	case basetypes.TupleValue:
		return val.Elements(), nil
	case basetypes.ListValue:
		return val.Elements(), nil
	case basetypes.SetValue:
		return val.Elements(), nil
	default:
		return nil, fmt.Errorf("keys must be a list of PEM strings or JWK objects, got %T", v)
	}
}

// thumbprintHash maps a hash name to a crypto.Hash for RFC 7638 thumbprints.
func thumbprintHash(name string) (crypto.Hash, bool) {
	switch name {
	case "SHA-256", "sha-256", "SHA256", "sha256":
		return crypto.SHA256, true
	case "SHA-1", "sha-1", "SHA1", "sha1":
		return crypto.SHA1, true
	case "SHA-384", "sha-384", "SHA384", "sha384":
		return crypto.SHA384, true
	case "SHA-512", "sha-512", "SHA512", "sha512":
		return crypto.SHA512, true
	default:
		return 0, false
	}
}
