package jobs

import (
	"arvika.pulcy.com/pulcy/deployit/units"
)

type Service interface {
	Name() string
	Units(currentScaleGroup uint8) ([]units.Unit, error)
}
