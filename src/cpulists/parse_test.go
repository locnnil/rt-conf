package cpulists

import (
	"reflect"
	"testing"
)

func TestParseCPUListsHappy(t *testing.T) {
	type test struct {
		input  string
		tCores int
		output CPUs
	}

	var tst = []test{
		// single CPU
		{
			"all",
			4,
			CPUs{0: true, 1: true, 2: true, 3: true},
		},
		{
			"all",
			2,
			CPUs{0: true, 1: true},
		},
		{
			"4",
			8,
			CPUs{4: true},
		},
		// 3 single CPUs
		{
			"4,5,9",
			10,
			CPUs{4: true, 5: true, 9: true},
		},
		// CPU range
		{
			"0-7",
			10,
			CPUs{0: true, 1: true, 2: true, 3: true, 4: true,
				5: true, 6: true, 7: true},
		},
		// two CPUs ranges
		{
			"0-2,4-7",
			10,
			CPUs{0: true, 1: true, 2: true, 4: true,
				5: true, 6: true, 7: true},
		},
		// CPU range + single CPU
		{
			"0-2,3",
			4,
			CPUs{0: true, 1: true, 2: true, 3: true},
		},
		// Formated CPU list
		{
			"0-20:2/5",
			24,
			CPUs{0: true, 1: true, 5: true, 6: true,
				10: true, 11: true, 15: true, 16: true, 20: true},
		},
		// Formated CPU list + a single CPU
		{
			"0-20:2/5,23",
			24,
			CPUs{0: true, 1: true, 5: true, 6: true, 10: true,
				11: true, 15: true, 16: true, 20: true, 23: true},
		},
	}

	for _, tt := range tst {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseForCPUs(tt.input, tt.tCores)
			if err != nil {
				t.Fatalf("ParseCPUs failed: %v", err)
			}
			if len(res) != len(tt.output) {
				t.Fatalf("expected %v, got %v", tt.output, res)
			}
			if !reflect.DeepEqual(res, tt.output) {
				t.Fatalf("expected %v, got %v", tt.output, res)
			}
		})
	}
}

func TestParseCPUListsUnhappy(t *testing.T) {
	type test struct {
		input  string
		tCores int
		err    string
	}

	var tst = []test{
		{
			"al",
			2,
			"invalid CPU: al",
		},
		{
			"alll",
			2,
			"invalid CPU: alll",
		},
		{
			"0-all",
			4,
			"invalid end of range: all",
		},
		{
			"all-n",
			4,
			"invalid start of range: all",
		},
		{
			"4",
			4,
			"CPU greater than total CPUs: 4",
		},
		{
			"a",
			4,
			"invalid CPU: a",
		},
		{
			"1-2-3",
			4,
			"invalid range: 1-2-3",
		},
		{
			"a-2",
			4,
			"invalid start of range: a",
		},
		{
			"1-a",
			4,
			"invalid end of range: a",
		},
		{
			"6-8",
			8,
			"end of range greater than total CPUs: 6-8",
		},
		{
			"5-2",
			8,
			"start of range greater than end: 5-2",
		},
		{
			"0--2:",
			4,
			"invalid range: 0--2",
		},
		{
			"0-:2",
			4,
			"invalid end of range: ",
		},
		{
			"0-2:",
			4,
			"invalid group size or used size: ",
		},
		{
			"0-2/8:10",
			8,
			"invalid end of range: 2/8",
		},
		{
			"a-2/8:10",
			8,
			"invalid start of range: a",
		},
		{
			"0-2a:10",
			8,
			"invalid end of range: 2a",
		},
		{
			"0-2:10",
			8,
			"invalid group size or used size: 10",
		},
		{
			"0-2:0/8",
			8,
			"used size must be at least 1, got: 0",
		},
		{
			"0-3:9/10",
			8,
			"used size greater than total CPUs: 9",
		},
	}

	for _, tt := range tst {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseForCPUs(tt.input, tt.tCores)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.err {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}
}

func TestParseWithFlagsHappy(t *testing.T) {
	isolcpuFlags := []string{"domain", "nohz", "managed_irq"}
	const max = 24
	var testCases = []struct {
		value string
		cpus  int
		flags []string
	}{
		// Test CPU list with range
		{"0-1", max, isolcpuFlags},
		{"nohz,0-1", max, isolcpuFlags},
		{"domain,0-1", max, isolcpuFlags},
		{"managed_irq,0-1", max, isolcpuFlags},

		// Test single CPU on CPU list
		{"0", max, isolcpuFlags},
		{"nohz,0", max, isolcpuFlags},
		{"domain,0", max, isolcpuFlags},
		{"managed_irq,0", max, isolcpuFlags},

		// Test comma separated CPU list
		{"0,n", max, isolcpuFlags},
		{"nohz,0,n", max, isolcpuFlags},
		{"domain,0,n", max, isolcpuFlags},
		{"managed_irq,0,n", max, isolcpuFlags},

		// Test comma separated CPU list
		{"0,n", max, isolcpuFlags},
		{"nohz,0,n", max, isolcpuFlags},
		{"domain,0,n", max, isolcpuFlags},
		{"managed_irq,0,n", max, isolcpuFlags},
	}

	for _, tc := range testCases {
		t.Run("TestValidationWithFlags", func(t *testing.T) {
			_, _, err := ParseWithFlagsForCPUs(tc.value, tc.flags, tc.cpus)
			if err != nil {
				t.Fatalf("Failed ValidateListWithFlags: %v", err)
			}
		})
	}
}
