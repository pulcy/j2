package jobs

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type Generator struct {
	job                 *Job
	groups              []TaskGroupName
	files               []string
	unitNames           []string
	tmpDir              string
	currentScalingGroup uint8
}

func newGenerator(job *Job, groups []TaskGroupName, currentScalingGroup uint8) *Generator {
	tmpDir, err := ioutil.TempDir("", "deployit")
	if err != nil {
		panic(err.Error())
	}
	return &Generator{
		job:                 job,
		groups:              groups,
		currentScalingGroup: currentScalingGroup,
		tmpDir:              tmpDir,
	}
}

func (g *Generator) WriteTmpFiles() error {
	files := []string{}
	unitNames := []string{}
	for _, tg := range g.job.Groups {
		if !g.include(tg.Name) {
			// We do not want this task group now
			continue
		}
		units, err := tg.createUnits(g.currentScalingGroup)
		if err != nil {
			return maskAny(err)
		}
		for _, unit := range units {
			content := unit.Render()
			unitName := unit.FullName
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

// Should the group with given name be generated?
func (g *Generator) include(groupName TaskGroupName) bool {
	if len(g.groups) == 0 {
		// include all
		return true
	}
	for _, n := range g.groups {
		if n == groupName {
			return true
		}
	}
	return false
}
