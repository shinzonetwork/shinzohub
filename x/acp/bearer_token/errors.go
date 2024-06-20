package bearer_token

import (
	"fmt"
	"time"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var ErrInvalidBearerToken = types.ErrAcpInput.Wrap("invalid bearer token")
var ErrMsgUnauthorized = fmt.Errorf("bearer policy msg: authorized_account doesn't match: %w", types.ErrNotAuthorized)

var ErrInvalidIssuer = fmt.Errorf("invalid issuer: expected did: %w", ErrInvalidBearerToken)
var ErrInvalidAuhtorizedAccount = fmt.Errorf("invalid authorized_account: expects SourceHub address: %w", ErrInvalidBearerToken)
var ErrTokenExpired = fmt.Errorf("token expired: %w", ErrInvalidBearerToken)
var ErrMissingClaim = fmt.Errorf("requried claim not found: %w", ErrInvalidBearerToken)
var ErrJSONSerializationUnsupported = fmt.Errorf("JWS JSON Serialization not supported: %w", ErrInvalidBearerToken)

func newErrTokenExpired(expiresUnix int64, nowUnix int64) error {
	expires := time.Unix(expiresUnix, 0).Format(time.DateTime)
	now := time.Unix(nowUnix, 0).Format(time.DateTime)

	return fmt.Errorf("expired %v: now %v: %w", expires, now, ErrTokenExpired)
}

func newErrMissingClaim(claim string) error {
	return fmt.Errorf("claim %v: %w", claim, ErrMissingClaim)
}
