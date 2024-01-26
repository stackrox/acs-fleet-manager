package ocm

// IDGenerator NOTE: the current mock generation exports to a _test file, if in the future this should be made public, consider
// moving the type into a ocmtest package.

// IDGenerator interface for string ID generators.
//
//go:generate moq -rm -out mocks/id_generator_moq.go -pkg mocks . IDGenerator
type IDGenerator interface {
	// Generate create a new string ID.
	Generate() string
}
