package test

import (
	"fmt"
	"os"

	"github.com/initialed85/dspo/pkg/process"
)

const (
	scriptTemplate = `
#!/bin/bash

set -e

if cat /tmp/%v_probe_harness.tmp | grep '%v_probe_ready: true'; then
	exit 0
fi

exit 1
`
)

type ProbeHarness struct {
	name string
}

func NewProbeHarness(name string) *ProbeHarness {
	p := ProbeHarness{
		name: name,
	}

	err := os.WriteFile(
		fmt.Sprintf("/tmp/%v_probe_test.sh", p.name),
		[]byte(fmt.Sprintf(scriptTemplate, p.name, p.name)),
		0o644,
	)
	if err != nil {
		panic(err)
	}

	chmodProcess := process.Run(
		"/bin/bash",
		fmt.Sprintf("chmod +x /tmp/%v_probe_test.sh", p.name),
		nil,
		false,
		nil,
		nil,
	)
	err = chmodProcess.Wait()
	if err != nil {
		panic(err)
	}

	p.SetNotReady()

	return &p
}

func (p *ProbeHarness) GetExecutablePath() string {
	return fmt.Sprintf("/tmp/%v_probe_test.sh", p.name)
}

func (p *ProbeHarness) SetReady() {
	err := os.WriteFile(
		fmt.Sprintf("/tmp/%v_probe_harness.tmp", p.name),
		[]byte(fmt.Sprintf("%v_probe_ready: true\n", p.name)),
		0o644,
	)
	if err != nil {
		panic(err)
	}
}

func (p *ProbeHarness) SetNotReady() {
	err := os.WriteFile(
		fmt.Sprintf("/tmp/%v_probe_harness.tmp", p.name),
		[]byte(fmt.Sprintf("%v_probe_ready: false\n", p.name)),
		0o644,
	)
	if err != nil {
		panic(err)
	}
}
