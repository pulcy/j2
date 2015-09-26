package services

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type Generator struct {
	serviceGroup        *ServiceGroup
	service             string
	files               []string
	unitNames           []string
	tmpDir              string
	currentScalingGroup uint8
}

func newGenerator(serviceGroup *ServiceGroup, service string, currentScalingGroup uint8) *Generator {
	tmpDir, err := ioutil.TempDir("", "deployit")
	if err != nil {
		panic(err.Error())
	}
	return &Generator{
		serviceGroup:        serviceGroup,
		service:             service,
		currentScalingGroup: currentScalingGroup,
		tmpDir:              tmpDir,
	}
}

func (g *Generator) WriteTmpFiles() error {
	files := []string{}
	unitNames := []string{}
	for _, s := range g.serviceGroup.Services {
		if g.service != "" && g.service != s.Name() {
			// We do not want this service
			continue
		}
		units, err := s.Units(g.currentScalingGroup)
		if err != nil {
			return maskAny(err)
		}
		for _, unit := range units {
			content := unit.Render()
			unitName := unit.FullName()
			path := filepath.Join(g.tmpDir, unitName)
			err := ioutil.WriteFile(path, []byte(content), 0666)
			if err != nil {
				return maskAny(err)
			}
			files = append(files, path)
			unitNames = append(unitNames, unitName)
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

func (g *Generator) TmpDir() string {
	return g.tmpDir
}
