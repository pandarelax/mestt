//go:build !fyne

package gui

import "fmt"

func RunFyne() error {
	return fmt.Errorf("fyne GUI build is disabled; rebuild with '-tags fyne'")
}
