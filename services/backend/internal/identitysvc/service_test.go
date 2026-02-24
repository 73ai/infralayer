package identitysvc_test

import (
	"testing"

	"github.com/73ai/infralayer/services/backend/identitytest"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/identitysvctest"
)

func TestService(t *testing.T) {
	identitytest.Ensure(t, identitysvctest.NewConfig().Fixture())
}
