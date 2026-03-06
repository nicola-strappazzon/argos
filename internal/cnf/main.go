package cnf

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads key=value pairs from the given section in a CNF-style file.
// Returns an error if the file cannot be opened or the section is not found.
func Load(path, section string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	result := map[string]string{}
	inSection := false
	found := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = line[1:len(line)-1] == section
			if inSection {
				found = true
			}
			continue
		}

		if !inSection {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if !found {
		return nil, fmt.Errorf("section [%s] not found in %s", section, path)
	}

	return result, nil
}
