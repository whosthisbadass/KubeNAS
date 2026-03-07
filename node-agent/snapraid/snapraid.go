package snapraid

import "github.com/kubenas/kubenas/node-agent/pkg/parity"

func Sync(configPath string) error { return parity.RunOperation("sync", configPath) }
func Scrub(configPath string) error { return parity.RunOperation("scrub", configPath) }
func Fix(configPath string) error { return parity.RunOperation("fix", configPath) }
func Status(configPath string) (string, error) { return parity.Status(configPath) }
