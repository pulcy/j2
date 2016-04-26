// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/units"
)

type GeneratorConfig struct {
	Groups              []TaskGroupName
	CurrentScalingGroup uint
	DockerOptions       cluster.DockerOptions
	FleetOptions        cluster.FleetOptions
}

type Generator struct {
	job *Job
	GeneratorConfig
	files     []string
	unitNames []string
	tmpDir    string
}

type Images struct {
	VaultMonkey string // Docker image name of vault-monkey
	Wormhole    string // Docker image name of wormhole
	Alpine      string // Docker image name of alpine linux
	CephVolume  string // Docker image name of ceph-volume
}

var (
	defaultTmpDir string
)

func init() {
	var err error
	defaultTmpDir, err = ioutil.TempDir("", "j2")
	if err != nil {
		panic(err.Error())
	}
}

func newGenerator(job *Job, config GeneratorConfig) *Generator {
	return &Generator{
		job:             job,
		GeneratorConfig: config,
		tmpDir:          defaultTmpDir,
	}
}

type generatorContext struct {
	ScalingGroup  uint
	InstanceCount int
	Images
	DockerOptions cluster.DockerOptions
	FleetOptions  cluster.FleetOptions
}

// NewTmpDir sets up a new temporary directory for the generated files.
// Used for testing only.
func (g *Generator) NewTmpDir() {
	var err error
	os.RemoveAll(g.tmpDir)
	g.tmpDir, err = ioutil.TempDir("", "j2")
	if err != nil {
		panic(err.Error())
	}
}

func (g *Generator) WriteTmpFiles(ctx units.RenderContext, images Images, instanceCount int) error {
	files := []string{}
	unitNames := []string{}
	maxCount := g.job.MaxCount()
	if err := os.MkdirAll(g.tmpDir, 0755); err != nil {
		return maskAny(err)
	}
	for scalingGroup := uint(1); scalingGroup <= maxCount; scalingGroup++ {
		if g.CurrentScalingGroup != 0 && g.CurrentScalingGroup != scalingGroup {
			continue
		}
		for _, tg := range g.job.Groups {
			if !g.include(tg.Name) {
				// We do not want this task group now
				continue
			}
			genCtx := generatorContext{
				ScalingGroup:  scalingGroup,
				InstanceCount: instanceCount,
				Images:        images,
				DockerOptions: g.DockerOptions,
				FleetOptions:  g.FleetOptions,
			}
			unitChains, err := tg.createUnits(genCtx)
			if err != nil {
				return maskAny(err)
			}
			for _, chain := range unitChains {
				for _, unit := range chain {
					content := unit.Render(ctx)
					unitName := unit.FullName
					path := filepath.Join(g.tmpDir, unitName)
					if _, err := os.Stat(path); err == nil {
						return maskAny(fmt.Errorf("'%s' already exists"))
					}
					err := ioutil.WriteFile(path, []byte(content), 0644)
					if err != nil {
						return maskAny(err)
					}
					files = append(files, path)
					unitNames = append(unitNames, unitName)
				}
			}
		}
	}
	g.files = files
	g.unitNames = unitNames
	return nil
}

func (g *Generator) RemoveTmpFiles() error {
	for _, path := range g.files {
		err := os.Remove(path)
		if err != nil {
			return maskAny(err)
		}
	}
	os.Remove(g.tmpDir) // If this fails we don't care
	return nil
}

func (g *Generator) FileNames() []string {
	return g.files
}

func (g *Generator) UnitNames() []string {
	return g.unitNames
}

// Should the group with given name be generated?
func (g *Generator) include(groupName TaskGroupName) bool {
	if len(g.Groups) == 0 {
		// include all
		return true
	}
	for _, n := range g.Groups {
		if n == groupName {
			return true
		}
	}
	return false
}
