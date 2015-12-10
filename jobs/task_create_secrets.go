package jobs

import (
	"strconv"

	"arvika.pulcy.com/pulcy/deployit/units"
)

// createSecretsUnit creates a unit used to extract secrets from vault
func (t *Task) createSecretsUnit(ctx generatorContext) (*units.Unit, error) {
	execStart := []string{"TODO"}
	unit := &units.Unit{
		Name:         t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription("Secrets", ctx.ScalingGroup),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	unit.ExecOptions.IsOneshot = true
	unit.ExecOptions.ExecStopPost = []string{
	// TODO cleanup
	//fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	}
	unit.FleetOptions.IsGlobal = t.group.Global
	if t.group.IsScalable() && ctx.InstanceCount > 1 {
		unit.FleetOptions.Conflicts(t.unitName(unitKindSecrets, "*") + ".service")
	}

	// Service dependencies
	// Requires=
	//main.ExecOptions.Require("flanneld.service")
	if requires, err := t.createMainRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.Require(requires...)
	}
	// After=...
	if after, err := t.createMainAfter(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.After(after...)
	}

	return unit, nil
}
