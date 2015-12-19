package jobs

import (
	"strconv"

	"arvika.pulcy.com/pulcy/deployit/units"
)

// createTimerUnit
func (t *Task) createTimerUnit(ctx generatorContext) (*units.Unit, error) {
	if t.Timer == "" {
		return nil, nil
	}
	unit := &units.Unit{
		Name:         t.unitName(unitKindTimer, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindTimer, strconv.Itoa(int(ctx.ScalingGroup))) + ".timer",
		Description:  t.unitDescription("Timer", ctx.ScalingGroup),
		Type:         "timer",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(),
		FleetOptions: units.NewFleetOptions(),
	}
	unit.ExecOptions.OnCalendar = t.Timer
	unit.ExecOptions.Unit = t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service"

	return unit, nil
}
