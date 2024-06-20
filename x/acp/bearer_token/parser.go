package bearer_token

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-jose/go-jose/v3"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
)

var requriedClaims = []string{
	IssuedAtClaim,
	IssuerClaim,
	AuthorizedAccountClaim,
	ExpiresClaim,
}

// parseValidateJWS processes a JWS Bearer token by unmarshaling it and verifying its signature.
// Returns a BearerToken is the JWS is valid and the signature matches the IssuerID
//
// Note: the JWS must be compact serialized, JSON serialization will be rejected as a conservative
// security measure against unprotected header attacks.
func parseValidateJWS(ctx context.Context, resolver did.Resolver, bearerJWS string) (BearerToken, error) {
	bearerJWS = strings.TrimLeft(bearerJWS, " \n\t\r")
	if strings.HasPrefix(bearerJWS, "{") {
		return BearerToken{}, ErrJSONSerializationUnsupported
	}

	jws, err := jose.ParseSigned(bearerJWS)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed parsing jws: %v: %w", err, ErrInvalidBearerToken)
	}

	payloadBytes := jws.UnsafePayloadWithoutVerification()
	bearer, err := unmarshalJWSPayload(payloadBytes)
	if err != nil {
		return BearerToken{}, err
	}

	err = validateBearerTokenValues(&bearer)
	if err != nil {
		return BearerToken{}, err
	}

	did := bearer.IssuerID
	doc, err := resolver.Resolve(ctx, did)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed to resolve actor did: %v: %w", err, ErrInvalidBearerToken)
	}
	// TODO this should technically be Authentication
	if len(doc.VerificationMethod) == 0 {
		return BearerToken{}, fmt.Errorf("resolved actor did does not contain any verification methods: %w", ErrInvalidBearerToken)
	}

	method := doc.VerificationMethod[0]
	jwkRaw, err := json.Marshal(method.PublicKeyJWK)
	if err != nil {
		return BearerToken{}, fmt.Errorf("error verifying signature: jwk marshal: %v: %w", err, ErrInvalidBearerToken)
	}

	jwk := jose.JSONWebKey{}
	err = jwk.UnmarshalJSON(jwkRaw)
	if err != nil {
		return BearerToken{}, fmt.Errorf("error verifying signature: jwk unmarshal: %v: %w", err, ErrInvalidBearerToken)
	}

	_, err = jws.Verify(jwk)
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not verify actor signature for jwk: %v: %w", err, ErrInvalidBearerToken)
	}

	return bearer, nil
}

// unmarshalJWSPayload unamrashals the JWS bytes into a BearerToken.
//
// The unarmshaling is strict, meaning that if the json object did not contain *all*
// required claims, it returns an error.
func unmarshalJWSPayload(payload []byte) (BearerToken, error) {
	obj := make(map[string]any)
	err := json.Unmarshal(payload, &obj)
	if err != nil {
		return BearerToken{}, err
	}

	for _, claim := range requriedClaims {
		_, ok := obj[claim]
		if !ok {
			return BearerToken{}, newErrMissingClaim(claim)
		}
	}

	token := BearerToken{}
	err = json.Unmarshal(payload, &token)
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not unmarshal payload: %v: %w", err, ErrInvalidBearerToken)
	}
	return token, nil
}
