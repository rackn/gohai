//+build !linux

package net

func (i *Interface) Fill() error {
	return nil
}