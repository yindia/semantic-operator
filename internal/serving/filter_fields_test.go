package serving

import (
	"testing"

	"github.com/kubeflow/semantic-operator/internal/governance"
	"github.com/kubeflow/semantic-operator/internal/planner/expr"
)

// Filters accept any modeled field, not just declared dimensions, so the
// filter listing must include employee_count even though list_dimensions
// leaves it out.
func TestListFilterFieldsIncludesNonDimensions(t *testing.T) {
	svc := &Service{Store: NewStore()}

	fields, err := svc.ListFilterFields(governedModel(), governance.Single("admin"))
	if err != nil {
		t.Fatalf("ListFilterFields: %v", err)
	}
	want := []FilterFieldInfo{
		{Name: "store.s_state", IsDimension: true},
		{Name: "store.manager_ssn", IsDimension: true},
		{Name: "store.employee_count"},
	}
	if len(fields) != len(want) {
		t.Fatalf("admin should see every field, got %+v", fields)
	}
	for i := range want {
		if fields[i].Name != want[i].Name || fields[i].IsDimension != want[i].IsDimension {
			t.Fatalf("field %d = %+v, want %+v", i, fields[i], want[i])
		}
	}
}

// A field the role may not read is left out, the same as in list_dimensions.
func TestListFilterFieldsHidesDeniedFields(t *testing.T) {
	svc := &Service{Store: NewStore()}

	fields, err := svc.ListFilterFields(governedModel(), governance.Single("analyst"))
	if err != nil {
		t.Fatalf("ListFilterFields: %v", err)
	}
	for _, f := range fields {
		if f.Name == "store.manager_ssn" {
			t.Fatalf("analyst can't read manager_ssn, but it was listed: %+v", fields)
		}
	}
	if len(fields) != 2 {
		t.Fatalf("analyst should see s_state and employee_count, got %+v", fields)
	}

	if _, err := svc.ListFilterFields(governedModel(), governance.Single("stranger")); err == nil {
		t.Fatal("a role with no policy should be refused")
	}
}

// Every listed field must pass governance when used as a filter.
func TestListFilterFieldsAgreesWithEnforcement(t *testing.T) {
	svc := &Service{Store: NewStore()}
	m := governedModel()
	id := governance.Single("analyst")

	fields, err := svc.ListFilterFields(m, id)
	if err != nil {
		t.Fatalf("ListFilterFields: %v", err)
	}
	for _, f := range fields {
		ref := expr.FieldRef{Dataset: "store", Field: f.Name[len("store."):]}
		if _, err := governance.Authorize(m.Governance, id, []string{"revenue"}, []expr.FieldRef{ref}); err != nil {
			t.Fatalf("listed filter field %q is refused by Authorize: %v", f.Name, err)
		}
	}
}
