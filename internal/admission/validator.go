package admission

import (
	"context"

	dataplanevalidation "github.com/chever-john/apisix-operator/internal/validation/dataplane"
	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
)

type validator struct {
	dataplaneValidator *dataplanevalidation.Validator
}

func (v *validator) ValidateControlPlane(ctx context.Context, controlPlane apisixoperatorv1alpha1.ControlPlane) error {
	return nil
}

func (v *validator) ValidateDataPlane(ctx context.Context, dataPlane apisixoperatorv1alpha1.DataPlane) error {
	return v.dataplaneValidator.Validate(&dataPlane)
}
