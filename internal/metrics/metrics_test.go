package metrics

import (
	"testing"
)

func TestIncrementFunctions(t *testing.T) {

	IncPVZCreated()
	IncReceptionCreated()
	IncReceptionCreated()
	IncProductAdded()
	IncProductAdded()
	IncProductAdded()

}
