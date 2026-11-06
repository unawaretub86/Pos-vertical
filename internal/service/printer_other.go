//go:build !windows

package service

import "context"

// Send is unsupported off Windows: the print queue only exists there.
func (w WindowsTransport) Send(ctx context.Context, data []byte) error {
	return ErrUnsupported
}
