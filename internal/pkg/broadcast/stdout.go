package broadcast

import (
	"encoding/json"
	"fmt"
	"os"
)

type StdOut struct{}

func (s StdOut) New() (Broadcast, error) {
	return s, nil
}

func (s StdOut) Send(message Story) error {
	res, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message to JSON failed: %w", err)
	}

	_, err = fmt.Fprintln(os.Stdout, string(res))
	if err != nil {
		return fmt.Errorf("failed to write message to stdout: %w", err)
	}

	return nil
}

func (s StdOut) Name() string {
	return "stdout"
}
