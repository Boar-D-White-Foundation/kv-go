package dummy_package

// These packages stay as dependencies of the root
import (
	_ "github.com/Boar-D-White-Foundation/kv-go/cli"        // keep
	_ "github.com/Boar-D-White-Foundation/kv-go/monitoring" // keep
	_ "github.com/Boar-D-White-Foundation/kv-go/node"       // keep
	_ "github.com/Boar-D-White-Foundation/kv-go/server"     // keep
	_ "github.com/Boar-D-White-Foundation/kv-go/tests"      // keep
)
