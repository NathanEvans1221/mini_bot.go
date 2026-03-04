package agent

import (
	"os"
)

const MaxMessageLength = 10000

type MemoryManager struct {
	MemoryPath string
}

func NewMemoryManager(memoryPath string) *MemoryManager {
	return &MemoryManager{
		MemoryPath: memoryPath,
	}
}

func (m *MemoryManager) Read() (string, error) {
	if _, err := os.Stat(m.MemoryPath); os.IsNotExist(err) {
		return "", nil
	}
	data, err := os.ReadFile(m.MemoryPath)
	return string(data), err
}

func (m *MemoryManager) Append(content string) error {
	cleaned := SanitizeInput(content)
	if len(cleaned) > MaxMessageLength {
		cleaned = cleaned[:MaxMessageLength]
	}
	f, err := os.OpenFile(m.MemoryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(cleaned + "\n"); err != nil {
		return err
	}
	return nil
}
