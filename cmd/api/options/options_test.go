package options_test

import (
	"testing"

	"github.com/adrianozp/gaardrail/cmd/api/options"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestOptions_DependencyGraphIsComplete(t *testing.T) {
	require.NoError(t, fx.ValidateApp(options.Options()))
}
