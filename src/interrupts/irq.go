package interrupts

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/canonical/rt-conf/src/cpu"
	"github.com/canonical/rt-conf/src/data"
	"github.com/canonical/rt-conf/src/helpers"
)

// NOTE: to be able to remap IRQs:
// CONFIG_REGMAP_IRQ=y must be present in the kernel config
// cat /boot/config-`uname -r` | grep CONFIG_REGMAP_IRQ

// /proc/interrupts man page:
// https://man7.org/linux/man-pages/man5/proc_interrupts.5.html

// OBS: There is no man page for /proc/irq

// NOTE: The Documentation for the procfs is available at:
// https://docs.kernel.org/filesystems/proc.html

// The Kernel space API irq_create_mapping_affinity() returns the IRQ number
// for a given interrupt line and affinity mask. This is used to map the
// interrupt line to the IRQ number.
// https://elixir.bootlin.com/linux/v6.13-rc2/source/kernel/irq/irqdomain.c#L834

type ProcIRQNumber struct {
	Number   int
	Affinity string
	CPUs     cpu.CPUs
}

type ProcInterrupts struct {
	Number int
	CPUs   cpu.CPUs
	Name   string
}

func ProcessIRQIsolation(cfg *data.InternalConfig) error {
	irqs, err := MapSystemIRQs()
	maxcpus := runtime.NumCPU()
	if err != nil {
		return fmt.Errorf("error mapping IRQs: %v", err)
	}

	isolCPUs := cfg.Data.Interrupts.IsolateCPU
	newAffinity := cfg.Data.Interrupts.IRQHandler
	if newAffinity == "" {
		var err error
		newAffinity, err = cpu.GenerateComplementCPUList(isolCPUs, maxcpus)
		if err != nil {
			return fmt.Errorf("error generating complement CPU list: %v", err)
		}
	}

	if err := RemapIRQAffinity(newAffinity, irqs); err != nil {
		return fmt.Errorf("error performing CPU isolation: %v", err)
	}

	return nil
}

func RemapIRQAffinity(newAffinity string, irq []ProcIRQNumber) error {
	maxcpus := runtime.NumCPU()
	fmt.Println("Total CPUs:", maxcpus)
	for _, i := range irq {
		f := fmt.Sprintf("/proc/irq/%d/smp_affinity_list", i.Number)
		err := helpers.WriteToFile(f, newAffinity)
		if err == nil {
			continue
		}
		// Managed IRQs are not allowed to be written to from userspace.
		// It fails with "input/output error"
		if !strings.Contains(err.Error(), "input/output error") {
			return fmt.Errorf("error writing to %s: %v", f, err)
		}
		log.Printf("Managed IRQ %d, skipped\n", i.Number)
	}
	return nil
}

// Map IRQs from /proc/irq
func MapSystemIRQs() ([]ProcIRQNumber, error) {
	dirEntries, err := os.ReadDir("/proc/irq")
	if err != nil {
		return nil, fmt.Errorf("error reading /proc/irq directory: %v", err)
	}

	var irqNumbers []ProcIRQNumber
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}

		irqNumber, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		affinityFile := fmt.Sprintf("/proc/irq/%d/smp_affinity_list", irqNumber)
		bytes, err := os.ReadFile(affinityFile)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", affinityFile, err)
		}

		affinity := strings.TrimSpace(string(bytes))

		cpus, err := cpu.ParseCPUs(affinity, runtime.NumCPU())
		if err != nil {
			return nil, fmt.Errorf("error parsing CPUs for IRQ %d: %v", irqNumber, err)
		}

		fmt.Println("\nIRQ:", irqNumber, "CPUs:", cpus)

		irqNumbers = append(irqNumbers, ProcIRQNumber{
			Number:   irqNumber,
			Affinity: affinity,
			CPUs:     cpus,
		})
	}

	return irqNumbers, nil
}

// Since there are some differences (not sure why) on the presence of IRQs
// on /proc/interrupts and /proc/irq it was decidede to have the
// capability to map both of them.
// But for now, we are going to use the mapping from /proc/irq

// Map interrupts from /proc/interrupts
func MapIRQs() ([]ProcInterrupts, error) {
	procfs, err := os.Open("/proc/interrupts")
	if err != nil {
		return nil, fmt.Errorf("error opening /proc/interrupts: %v", err)
	}
	defer procfs.Close()

	var irqs []ProcInterrupts
	scanner := bufio.NewScanner(procfs)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		irqNumber, err := strconv.Atoi(strings.TrimSuffix(fields[0], ":"))
		if err != nil {
			continue
		}

		irqName := fields[len(fields)-1]
		cpus, err := getCPUsForIRQ(irqNumber)
		if err != nil {
			return nil, fmt.Errorf("error getting CPUs for IRQ %d: %v", irqNumber, err)
		}

		irqs = append(irqs, ProcInterrupts{
			Number: irqNumber,
			CPUs:   cpus,
			Name:   irqName,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /proc/interrupts: %v", err)
	}

	return irqs, nil
}

func getCPUsForIRQ(irqN int) (cpu.CPUs, error) {
	procfs := fmt.Sprintf("/proc/irq/%d/smp_affinity_list", irqN)
	file, err := os.Open(procfs)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", procfs, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		cpus, err := cpu.ParseCPUs(scanner.Text(), runtime.NumCPU())
		if err != nil {
			err := fmt.Errorf("error parsing CPUs for IRQ %d: %v", irqN, err)
			return nil, err
		}
		return cpus, nil
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading", procfs, ":", err)
	}

	return nil, err
}

// FUTURE IDEAS:

// On /proc/interrupts
// Monitor:
// TRM - checking for thermal throttling on the CPUs
