package evm

import "os"

func writeFileBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}
