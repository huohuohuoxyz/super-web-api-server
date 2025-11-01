package main

import (
	"os/exec"
	"strings"
)

func GetAllDrives() ([]string, error) {
	cmd := exec.Command("cmd", "/C", "wmic logicaldisk get name")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var drives []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 只考虑包含":"的行，表示有效的磁盘分区
		if strings.HasSuffix(line, ":") {
			drives = append(drives, line)
		}
	}
	return drives, nil
}
