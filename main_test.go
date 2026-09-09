package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/argoproj/argo-rollouts/metricproviders/plugin/rpc"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	goPlugin "github.com/hashicorp/go-plugin"
)

// Exercise the shipped entry point over the controller's RPC protocol, including
// the handshake that the in-process reattach tests do not validate.
func TestExecutableHandshake(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "coralogix-plugin")
	// The protocol test also runs from source archives and isolated worktrees;
	// VCS stamping is unrelated to the executable behavior under test.
	if output, err := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, output)
	}
	client := goPlugin.NewClient(&goPlugin.ClientConfig{
		HandshakeConfig: goPlugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "ARGO_ROLLOUTS_RPC_PLUGIN",
			MagicCookieValue: "metricprovider",
		},
		Plugins: map[string]goPlugin.Plugin{
			"RpcMetricProviderPlugin": &rpc.RpcMetricProviderPlugin{},
		},
		Cmd: exec.Command(binary),
	})
	t.Cleanup(client.Kill)
	protocol, err := client.Client()
	if err != nil {
		t.Fatalf("connect to executable: %v", err)
	}
	raw, err := protocol.Dispense("RpcMetricProviderPlugin")
	if err != nil {
		t.Fatalf("dispense metric provider: %v", err)
	}
	provider, ok := raw.(rpc.MetricProviderPlugin)
	if !ok {
		t.Fatal("executable does not implement the upstream metric provider interface")
	}
	if err := provider.InitPlugin(); err.HasError() {
		t.Fatalf("initialize: %s", err.Error())
	}
	if got := provider.Type(); got != "RPCPlugin" {
		t.Fatalf("provider type = %q", got)
	}
	// Invalid configuration must fail before any external request is made.
	measurement := provider.Run(&v1alpha1.AnalysisRun{}, v1alpha1.Metric{
		Provider: v1alpha1.MetricProvider{
			Plugin: map[string]json.RawMessage{"argoproj-labs/coralogix-metric-plugin": json.RawMessage(`{"query":`)},
		},
	})
	if measurement.Phase != v1alpha1.AnalysisPhaseError {
		t.Fatalf("invalid configuration returned phase %s", measurement.Phase)
	}
}
