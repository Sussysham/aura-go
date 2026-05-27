package tui

import (
	"os"
)

func ReadInput(out chan []byte) {
	var buf [10]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return
		}
		out <- append([]byte{}, buf[:n]...)
	}
}
