package services

import (
	"arvika.pulcy.com/iggi/deployit/units"
)

type Service interface {
	Name() string
	Units(currentScaleGroup uint8) ([]units.Unit, error)
}
