/*
JWT / JWS compact-token functions: jwt_sign, jwt_decode, jwt_verify.

All three are pure and deterministic. jwt_sign never reads the clock: `exp`, `iat`, and `nbf`, if present, are whatever the caller put in `claims`. jwt_verify only touches time when the caller supplies `options.now`; with no `now` it validates the signature alone and stays clock-free.
*/

package cryptography

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// jwtTokenMaxBytes bounds the token jwt_decode / jwt_verify will parse. 1 MiB is far above any realistic JWT (typical tokens are well under 8 KiB) while still bounding the base64url decode and JSON parse work an adversarial input can force.
const jwtTokenMaxBytes = 1 * 1024 * 1024

// --- jwt_sign ---------------------------------------------------------------

var _ function.Function = (*JWTSignFunction)(nil)

//go:embed descriptions/jwt_sign.md
var jwtSignDescription string

type JWTSignFunction struct{}

func NewJWTSignFunction() function.Function { return &JWTSignFunction{} }

func (f *JWTSignFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwt_sign"
}

func (f *JWTSignFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Mint a compact JWS / JWT (deterministic; HS/ES256/EdDSA/RS families)",
		MarkdownDescription: jwtSignDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "claims", Description: "The JWT payload as an object. `exp`, `iat`, and `nbf` if present are used verbatim; this function never derives them from the clock."},
			function.StringParameter{Name: "algorithm", Description: "Signing algorithm: HS256, HS384, HS512 (key is the shared secret bytes); ES256 (key is a PEM EC P-256 private key); EdDSA (key is a PEM Ed25519 private key); RS256, RS384, RS512 (key is a PEM RSA private key)."},
			function.StringParameter{Name: "key", Description: "The signing key: the raw secret bytes for HS*, or a PEM-encoded private key for ES256 / EdDSA / RS*."},
		},
		VariadicParameter: function.DynamicParameter{Name: "options", Description: "Optional object of extra JWS header fields to merge in (for example `kid`, `typ`, `cty`). `alg` is always set from `algorithm`. Pass at most one."},
		Return:            function.StringReturn{},
	}
}

func (f *JWTSignFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var claims types.Dynamic
	var algorithm, key string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &claims, &algorithm, &key, &optsArgs))
	if resp.Error != nil {
		return
	}

	unknown := hasUnknownValue(claims)
	for _, o := range optsArgs {
		if hasUnknownValue(o) {
			unknown = true
		}
	}
	if unknown {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	if _, ok := jwsAlgHash(algorithm); !ok {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("unsupported algorithm %q (want one of HS256, HS384, HS512, ES256, EdDSA, RS256, RS384, RS512)", algorithm))
		return
	}
	if len(optsArgs) > 1 {
		resp.Error = function.NewArgumentFuncError(3, "at most one options object may be provided")
		return
	}

	claimsGo, err := terraformToGoJSON(claims.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert claims: "+err.Error())
		return
	}
	if _, ok := claimsGo.(map[string]interface{}); !ok && claimsGo != nil {
		resp.Error = function.NewArgumentFuncError(0, "claims must be an object")
		return
	}

	header := map[string]interface{}{"alg": algorithm}
	if len(optsArgs) == 1 {
		optsGo, err := terraformToGoJSON(optsArgs[0].UnderlyingValue())
		if err != nil {
			resp.Error = function.NewArgumentFuncError(3, "failed to convert options: "+err.Error())
			return
		}
		if optsGo != nil {
			optsMap, ok := optsGo.(map[string]interface{})
			if !ok {
				resp.Error = function.NewArgumentFuncError(3, "options must be an object")
				return
			}
			for k, v := range optsMap {
				header[k] = v
			}
		}
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		resp.Error = function.NewFuncError("failed to encode JWS header: " + err.Error())
		return
	}
	payloadJSON, err := json.Marshal(claimsGo)
	if err != nil {
		resp.Error = function.NewFuncError("failed to encode JWT payload: " + err.Error())
		return
	}

	signingInput := b64uEncode(headerJSON) + "." + b64uEncode(payloadJSON)
	sig, err := signJWS([]byte(signingInput), algorithm, []byte(key))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(2, err.Error())
		return
	}
	token := signingInput + "." + b64uEncode(sig)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &token))
}

// --- jwt_decode -------------------------------------------------------------

var _ function.Function = (*JWTDecodeFunction)(nil)

//go:embed descriptions/jwt_decode.md
var jwtDecodeDescription string

type JWTDecodeFunction struct{}

func NewJWTDecodeFunction() function.Function { return &JWTDecodeFunction{} }

func (f *JWTDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwt_decode"
}

func (f *JWTDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a compact JWS / JWT into { header, payload } WITHOUT verifying the signature",
		MarkdownDescription: jwtDecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "token", Description: "A compact JWS / JWT (`header.payload.signature`)."},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JWTDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var token string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &token))
	if resp.Error != nil {
		return
	}
	if len(token) > jwtTokenMaxBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("token exceeds maximum supported length of %d bytes", jwtTokenMaxBytes))
		return
	}
	headerVal, payloadVal, _, ferr := splitAndDecodeJWT(token)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	obj, err := buildHeaderPayloadObject(headerVal, payloadVal)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// --- jwt_verify -------------------------------------------------------------

var _ function.Function = (*JWTVerifyFunction)(nil)

//go:embed descriptions/jwt_verify.md
var jwtVerifyDescription string

type JWTVerifyFunction struct{}

func NewJWTVerifyFunction() function.Function { return &JWTVerifyFunction{} }

func (f *JWTVerifyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jwt_verify"
}

func (f *JWTVerifyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Verify a compact JWS / JWT signature, returning { valid, header, payload }",
		MarkdownDescription: jwtVerifyDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "token", Description: "A compact JWS / JWT (`header.payload.signature`)."},
			function.StringParameter{Name: "key", Description: "The verification key: the shared secret bytes for HS*, or a PEM public key (`PUBLIC KEY`), certificate, or private key for ES256 / EdDSA / RS*."},
		},
		VariadicParameter: function.DynamicParameter{Name: "options", Description: "Optional object. `now` (unix-seconds number or RFC 3339 string) enables `exp` / `nbf` time validation; omit it to check the signature only. `algorithm` (string) pins the accepted alg and rejects any token whose header alg differs (guards against algorithm-substitution). Pass at most one."},
		Return:            function.DynamicReturn{},
	}
}

func (f *JWTVerifyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var token, key string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &token, &key, &optsArgs))
	if resp.Error != nil {
		return
	}
	if len(token) > jwtTokenMaxBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("token exceeds maximum supported length of %d bytes", jwtTokenMaxBytes))
		return
	}
	if len(optsArgs) > 1 {
		resp.Error = function.NewArgumentFuncError(2, "at most one options object may be provided")
		return
	}

	headerVal, payloadVal, parts, ferr := splitAndDecodeJWT(token)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	headerBytes, _ := b64uDecode(parts[0])
	var headerMap map[string]interface{}
	if err := json.Unmarshal(headerBytes, &headerMap); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "token header is not a JSON object: "+err.Error())
		return
	}
	alg, _ := headerMap["alg"].(string)
	if alg == "" {
		resp.Error = function.NewArgumentFuncError(0, "token header has no string `alg`")
		return
	}
	if _, ok := jwsAlgHash(alg); !ok {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("token header alg %q is not supported", alg))
		return
	}

	// Options: now (time validation) and algorithm (substitution guard).
	var nowSet bool
	var nowSeconds float64
	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok && !optsArgs[0].UnderlyingValue().IsNull() {
			resp.Error = function.NewArgumentFuncError(2, "options must be an object")
			return
		}
		if ok {
			attrs := obj.Attributes()
			if pinned, present, err := optionalStringAttr(attrs, "algorithm"); err != nil {
				resp.Error = function.NewArgumentFuncError(2, err.Error())
				return
			} else if present && pinned != alg {
				// Algorithm substitution: report invalid rather than verifying under an unexpected alg.
				f.setResult(ctx, resp, false, headerVal, payloadVal)
				return
			}
			if n, present, err := parseNowOption(attrs); err != nil {
				resp.Error = function.NewArgumentFuncError(2, err.Error())
				return
			} else if present {
				nowSet = true
				nowSeconds = n
			}
		}
	}

	sig, err := b64uDecode(parts[2])
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "token signature segment is not valid base64url: "+err.Error())
		return
	}
	signingInput := []byte(parts[0] + "." + parts[1])
	sigValid, err := verifyJWS(signingInput, sig, alg, []byte(key))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}

	valid := sigValid
	if valid && nowSet {
		payloadBytes, _ := b64uDecode(parts[1])
		timeValid, err := timeClaimsValid(payloadBytes, nowSeconds)
		if err != nil {
			resp.Error = function.NewArgumentFuncError(0, err.Error())
			return
		}
		valid = timeValid
	}

	f.setResult(ctx, resp, valid, headerVal, payloadVal)
}

func (f *JWTVerifyFunction) setResult(ctx context.Context, resp *function.RunResponse, valid bool, headerVal, payloadVal attr.Value) {
	attrTypes := map[string]attr.Type{
		"valid":   types.BoolType,
		"header":  headerVal.Type(nil),
		"payload": payloadVal.Type(nil),
	}
	attrVals := map[string]attr.Value{
		"valid":   types.BoolValue(valid),
		"header":  headerVal,
		"payload": payloadVal,
	}
	obj, diags := types.ObjectValue(attrTypes, attrVals)
	if diags.HasError() {
		resp.Error = function.NewFuncError("building result object: " + diagsToString(diags))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// --- shared helpers ---------------------------------------------------------

// splitAndDecodeJWT splits a compact JWS into its three segments and decodes the header and payload segments into Terraform values. The returned parts slice is the raw base64url segments. A malformed token yields an argument error on index 0.
func splitAndDecodeJWT(token string) (headerVal, payloadVal attr.Value, parts []string, ferr *function.FuncError) {
	parts = strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, nil, nil, function.NewArgumentFuncError(0, fmt.Sprintf("token must have exactly 3 dot-separated segments, got %d", len(parts)))
	}
	headerBytes, err := b64uDecode(parts[0])
	if err != nil {
		return nil, nil, nil, function.NewArgumentFuncError(0, "token header segment is not valid base64url: "+err.Error())
	}
	payloadBytes, err := b64uDecode(parts[1])
	if err != nil {
		return nil, nil, nil, function.NewArgumentFuncError(0, "token payload segment is not valid base64url: "+err.Error())
	}
	headerVal, err = jsonBytesToTerraform(headerBytes)
	if err != nil {
		return nil, nil, nil, function.NewArgumentFuncError(0, "token header is not valid JSON: "+err.Error())
	}
	payloadVal, err = jsonBytesToTerraform(payloadBytes)
	if err != nil {
		return nil, nil, nil, function.NewArgumentFuncError(0, "token payload is not valid JSON: "+err.Error())
	}
	return headerVal, payloadVal, parts, nil
}

func buildHeaderPayloadObject(headerVal, payloadVal attr.Value) (attr.Value, error) {
	attrTypes := map[string]attr.Type{
		"header":  headerVal.Type(nil),
		"payload": payloadVal.Type(nil),
	}
	attrVals := map[string]attr.Value{
		"header":  headerVal,
		"payload": payloadVal,
	}
	obj, diags := types.ObjectValue(attrTypes, attrVals)
	if diags.HasError() {
		return nil, fmt.Errorf("building result object: %s", diagsToString(diags))
	}
	return obj, nil
}

// optionalStringAttr fetches a string option; present is false when the key is absent or null.
func optionalStringAttr(attrs map[string]attr.Value, key string) (value string, present bool, err error) {
	v, ok := attrs[key]
	if !ok || v == nil || v.IsNull() {
		return "", false, nil
	}
	sv, ok := v.(basetypes.StringValue)
	if !ok {
		return "", false, fmt.Errorf("%q must be a string, got %T", key, v)
	}
	return sv.ValueString(), true, nil
}

// parseNowOption reads options.now as either a unix-seconds number or an RFC 3339 string and returns it as float seconds.
func parseNowOption(attrs map[string]attr.Value) (seconds float64, present bool, err error) {
	v, ok := attrs["now"]
	if !ok || v == nil || v.IsNull() {
		return 0, false, nil
	}
	switch nv := v.(type) {
	case basetypes.NumberValue:
		f, _ := nv.ValueBigFloat().Float64()
		return f, true, nil
	case basetypes.StringValue:
		t, err := time.Parse(time.RFC3339, nv.ValueString())
		if err != nil {
			return 0, false, fmt.Errorf("now is not a unix-seconds number or RFC 3339 string: %w", err)
		}
		return float64(t.Unix()), true, nil
	default:
		return 0, false, fmt.Errorf("now must be a number or RFC 3339 string, got %T", v)
	}
}

// timeClaimsValid checks exp / nbf against now (seconds). A token is time-valid when now < exp (if exp present) and now >= nbf (if nbf present).
func timeClaimsValid(payloadBytes []byte, now float64) (bool, error) {
	var payload map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(string(payloadBytes)))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return false, fmt.Errorf("payload is not a JSON object: %w", err)
	}
	if exp, ok, err := numericClaim(payload, "exp"); err != nil {
		return false, err
	} else if ok && now >= exp {
		return false, nil
	}
	if nbf, ok, err := numericClaim(payload, "nbf"); err != nil {
		return false, err
	} else if ok && now < nbf {
		return false, nil
	}
	return true, nil
}

func numericClaim(payload map[string]interface{}, key string) (float64, bool, error) {
	v, ok := payload[key]
	if !ok {
		return 0, false, nil
	}
	n, ok := v.(json.Number)
	if !ok {
		return 0, false, fmt.Errorf("claim %q must be a number", key)
	}
	f, err := n.Float64()
	if err != nil {
		return 0, false, fmt.Errorf("claim %q is not a valid number: %w", key, err)
	}
	return f, true, nil
}
