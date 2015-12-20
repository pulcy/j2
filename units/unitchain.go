package units

type UnitChain []*Unit

// Link adds Require & After attributes on the units in the chain
// in order to enfore the correct chain.
func (chain UnitChain) Link() {
	for i, u := range chain {
		if i+1 < len(chain) {
			// Forward requirement
			u.ExecOptions.Require(chain[i+1].FullName)
		}
		if i > 0 {
			// Backward requirement and after
			u.ExecOptions.Require(chain[i-1].FullName)
			u.ExecOptions.After(chain[i-1].FullName)
		}
	}
}
