//go:build !linux

package sessions

// NewManager returns an error on unsupported platforms.
func NewManager() (Manager, error) {
	return nil, ErrUnavailable
}
