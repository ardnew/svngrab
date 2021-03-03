package repo

// ExportMode represents the type of operation used to perform content retrieval
// from a VCS repository.
type ExportMode int

// Constant values of enumerated type ExportMode.
const (
	UpdateMode ExportMode = iota
	CheckoutMode
)

// String returns the string representation of the receiver ExportMode.
func (m ExportMode) String() string {
	return []string{"update", "checkout"}[m]
}
