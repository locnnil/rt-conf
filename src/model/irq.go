package model

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/canonical/rt-conf/src/cpulists"
)

// TODO: THis needs to be superseed in the unit tests
const (
	SysKernelIRQ = "/sys/kernel/irq"
	ProcIRQ      = "/proc/irq"
)

var readDir = func(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

type IRQTuning struct {
	CPUs   string    `yaml:"cpus"`
	Filter IRQFilter `yaml:"filter"`
}

func (c IRQTuning) Validate() error {
	err := c.Filter.Validate()
	if err != nil {
		return fmt.Errorf("IRQFilter validation failed: %v", err)
	}
	_, err = cpulists.Parse(c.CPUs)
	if err != nil {
		return fmt.Errorf("invalid cpus: %v", err)
	}
	return nil
}

type IRQFilter struct {
	Actions  string `yaml:"actions" validation:"regex"`
	ChipName string `yaml:"chip-name" validation:"regex"`
	Name     string `yaml:"name" validation:"regex"`
	Type     string `yaml:"type" validation:"regex"`
}

type IRQs struct {
	IsolateCPU string `yaml:"remove-from-cpus"`
	IRQHandler string `yaml:"handle-on-cpus"`
}

func (c IRQFilter) Validate() error {
	return Validate(c, c.validateIRQField)
}

// TODO: Validate mutual exclusive cpu lists

func (c IRQFilter) validateIRQField(name string, value string, tag string) error {
	switch {
	case tag == "cpulist":
		num, err := GetHigherIRQ()
		if err != nil {
			return err
		}
		_, err = cpulists.ParseForCPUs(value, num)
		if err != nil {
			return fmt.Errorf("on field %v: invalid irq list: %v", name,
				err)
		}
	case tag == "regex":
		_, err := regexp.Compile(value)
		if err != nil {
			return fmt.Errorf("on field %v: invalid regex: %v", name, err)
		}
	default:
		return fmt.Errorf("on field %v: invalid tag: %v", name, tag)
	}
	return nil
}

func GetHigherIRQ() (int, error) {
	files, err := readDir(SysKernelIRQ)
	if err != nil {
		return 0, err
	}
	var irqs []int
	for _, file := range files {
		num, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}
		if file.IsDir() {
			irqs = append(irqs, num)
		}
	}
	if len(irqs) == 0 {
		return 0, fmt.Errorf("no IRQs found")
	}
	bigger := irqs[0]
	for _, irq := range irqs {
		if irq > bigger {
			bigger = irq
		}
	}
	return bigger, nil
}
