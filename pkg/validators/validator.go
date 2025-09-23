package validators

type Validator interface {
	ValidateDid(str string) error
	ValidateDataFeedId(str string) error
}
