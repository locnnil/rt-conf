package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/canonical/rt-conf/src/data"
	"github.com/canonical/rt-conf/src/execute"
	"github.com/canonical/rt-conf/src/helpers"
)

// grubCfgTransformer handles transformations for /boot/grub/grub.cfg
type GrubCfgTransformer struct {
	FilePath string
	Pattern  *regexp.Regexp
	Params   []string
}

// grubDefaultTransformer handles transformations for /etc/default/grub
type GrubDefaultTransformer struct {
	FilePath string
	Pattern  *regexp.Regexp
	Params   []string
}

func (g *GrubCfgTransformer) TransformLine(line string) string {
	// Append each kernel command line parameter to the matched line
	for _, param := range g.Params {
		line += " " + param
	}
	return line
}

func (g *GrubCfgTransformer) GetFilePath() string {
	return g.FilePath
}

func (g *GrubCfgTransformer) GetPattern() *regexp.Regexp {
	return g.Pattern
}

func (g *GrubDefaultTransformer) TransformLine(line string) string {
	// TODO: Add functionality to avoid duplications of parameters

	// Extract existing parameters
	matches := g.Pattern.FindStringSubmatch(line)
	// Append new parameters
	updatedParams := strings.TrimSpace(matches[2] + " " + strings.Join(g.Params, " "))
	// Reconstruct the line with updated parameters
	return fmt.Sprintf(`%s%s%s`, matches[1], updatedParams, matches[3])
}

func (g *GrubDefaultTransformer) GetFilePath() string {
	return g.FilePath
}

func (g *GrubDefaultTransformer) GetPattern() *regexp.Regexp {
	return g.Pattern
}

// InjectToGrubFiles inject the kernel command line parameters to the grub files. /etc/default/grub
func UpdateGrub(cfg *data.InternalConfig) ([]string, error) {
	var msgs []string

	params, err := helpers.ReconstructKeyValuePairs(&cfg.Data.KernelCmdline)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct key-value pairs: %v", err)
	}

	grubDefault := &GrubDefaultTransformer{
		FilePath: cfg.GrubDefault.File,
		Pattern:  cfg.GrubDefault.Pattern,
		Params:   params,
	}

	if err := helpers.ProcessFile(grubDefault); err != nil {
		return nil, fmt.Errorf("error updating %s: %v", grubDefault.FilePath, err)
	}

	msgs = append(msgs, "Updated default grub file: "+grubDefault.FilePath+"\n")
	msgs = append(msgs, execute.GrubConclusion()...)

	return msgs, nil
}
