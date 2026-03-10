package snapraid

import "github.com/kubenas/kubenas/node-agent/pkg/parity"

func Sync(configPath string) error {
	_, err := parity.Sync(configPath)
	return err
}

func Scrub(configPath string) error {
	_, err := parity.Scrub(configPath, 100)
	return err
}

func Fix(configPath string, diskLabel string) error {
	_, err := parity.Fix(configPath, diskLabel)
	return err
}

func Status(configPath string) (string, error) {
	result, err := parity.Status(configPath)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
