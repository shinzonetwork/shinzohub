package validators

import "errors"

type RegistrarValidator struct{}

func (v *RegistrarValidator) ValidateDid(did string) error {
	if len(did) == 0 {
		return errors.New("did string must be non-empty")
	}

	return nil
}

func (v *RegistrarValidator) ValidateDataFeedId(dataFeedId string) error {
	if len(dataFeedId) == 0 {
		return errors.New("data feed id string must be non-empty")
	}

	return nil
}
