package mcp

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/planner"
	"github.com/kubeflow/semantic-operator/internal/serving"
)

// list_filter_fields returns fields that can be filtered on but not grouped
// by, which list_dimensions never shows.
func TestListFilterFieldsTool(t *testing.T) {
	blob, err := json.Marshal(planner.CompiledModel{
		Name: "retail", Version: "v1",
		DatasetOrder: []string{"store"},
		Datasets: map[string]*planner.CompiledDataset{
			"store": {
				Name: "store", FieldOrder: []string{"s_state", "employee_count"},
				Fields: map[string]*planner.CompiledField{
					"s_state":        {Name: "s_state", IsDimension: true},
					"employee_count": {Name: "employee_count", Type: "integer"},
				},
			},
		},
		Governance: &v1alpha1.GovernanceSpec{
			DefaultRole: "analyst",
			Roles:       []v1alpha1.RolePolicy{{Name: "analyst", AllowMetrics: []string{"*"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store := serving.NewStore()
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	serverT, clientT := sdk.NewInMemoryTransports()
	if _, err := NewServer(&serving.Service{Store: store}, "test", serving.StaticResolver()).Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "list_filter_fields", Arguments: map[string]any{"model": "retail"},
	})
	if err != nil {
		t.Fatalf("list_filter_fields: %v", err)
	}
	if res.IsError {
		t.Fatalf("list_filter_fields returned a tool error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out listFilterFieldsOut
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	if out.Model != "retail" || len(out.Fields) != 2 {
		t.Fatalf("want both fields of retail, got %s", raw)
	}
	if !out.Fields[0].IsDimension || out.Fields[1].Name != "store.employee_count" || out.Fields[1].IsDimension {
		t.Fatalf("unexpected fields: %s", raw)
	}
}
