package memory

import (
	"context"
	"log"

	"github.com/apparentlymart/terraform-provider-memory/internal/tfplugin6"
	"github.com/zclconf/go-cty/cty"
)

type Provider struct {
	tfplugin6.UnimplementedProviderServer

	logger          *log.Logger
	deferralAllowed bool
}

var _ tfplugin6.ProviderServer = (*Provider)(nil)

func NewProvider(logger *log.Logger) *Provider {
	return &Provider{
		logger:          logger,
		deferralAllowed: false, // might be set in ConfigureProvider
	}
}

// ApplyResourceChange implements tfplugin6.ProviderServer.
func (p *Provider) ApplyResourceChange(ctx context.Context, req *tfplugin6.ApplyResourceChange_Request) (*tfplugin6.ApplyResourceChange_Response, error) {
	plannedObj, err := memoryValFromProto(req.PlannedState)
	if err != nil {
		return &tfplugin6.ApplyResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to decode planned state value",
				"Planned state value is invalid", err,
			),
		}, nil
	}
	if plannedObj.GetAttr("value").IsKnown() {
		// If the planned value is known then we're not changing the
		// stored value and so we can just return what we were given.
		return &tfplugin6.ApplyResourceChange_Response{
			NewState: req.PriorState,
		}, nil
	}
	// Otherwise, we'll need to take the final new_value from
	// the configuration.
	configObj, err := memoryValFromProto(req.Config)
	if err != nil {
		return &tfplugin6.ApplyResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to decode configuration value",
				"Configuration value is invalid", err,
			),
		}, nil
	}
	newValue := configObj.GetAttr("new_value")
	if newValue.IsNull() {
		// We shouldn't get here because a null new_value should've
		// caused us to end up in the "planned value is null" branch
		// above, but we'll tolerate this anyway for robustness since
		// the rules for write-only attributes are considerably looser
		// than for other attributes.
		return &tfplugin6.ApplyResourceChange_Response{
			NewState: req.PriorState,
		}, nil
	}

	// If we get here then we _are_ going to change the stored value,
	// and so we need to construct the final object to return.
	newObj := cty.ObjectVal(map[string]cty.Value{
		"new_value": cty.NullVal(newValue.Type()), // read-only attribute, so always null in response
		"value":     newValue,
	})
	ret, err := memoryValToProto(newObj)
	if err != nil {
		return &tfplugin6.ApplyResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to serialize final state",
				"Could not serialize the updated state", err,
			),
		}, nil
	}
	return &tfplugin6.ApplyResourceChange_Response{
		NewState: ret,
	}, nil
}

// ConfigureProvider implements tfplugin6.ProviderServer.
func (p *Provider) ConfigureProvider(ctx context.Context, req *tfplugin6.ConfigureProvider_Request) (*tfplugin6.ConfigureProvider_Response, error) {
	if req.ClientCapabilities != nil && req.ClientCapabilities.DeferralAllowed {
		p.deferralAllowed = true
	}
	return &tfplugin6.ConfigureProvider_Response{}, nil
}

// GetFunctions implements tfplugin6.ProviderServer.
func (p *Provider) GetFunctions(context.Context, *tfplugin6.GetFunctions_Request) (*tfplugin6.GetFunctions_Response, error) {
	return &tfplugin6.GetFunctions_Response{}, nil
}

// GetMetadata implements tfplugin6.ProviderServer.
func (p *Provider) GetMetadata(context.Context, *tfplugin6.GetMetadata_Request) (*tfplugin6.GetMetadata_Response, error) {
	return &tfplugin6.GetMetadata_Response{
		ServerCapabilities: &tfplugin6.ServerCapabilities{},
	}, nil
}

// GetProviderSchema implements tfplugin6.ProviderServer.
func (p *Provider) GetProviderSchema(context.Context, *tfplugin6.GetProviderSchema_Request) (*tfplugin6.GetProviderSchema_Response, error) {
	return &tfplugin6.GetProviderSchema_Response{
		Provider: &tfplugin6.Schema{
			Block: &tfplugin6.Schema_Block{},
		},
		ResourceSchemas: map[string]*tfplugin6.Schema{
			"memory": {
				Block: &tfplugin6.Schema_Block{
					Attributes: []*tfplugin6.Schema_Attribute{
						{
							Name:      "new_value",
							Type:      []byte(`"dynamic"`),
							Optional:  true,
							WriteOnly: true,
						},
						{
							Name:     "value",
							Type:     []byte(`"dynamic"`),
							Computed: true,
						},
					},
				},
			},
		},
	}, nil
}

// PlanResourceChange implements tfplugin6.ProviderServer.
func (p *Provider) PlanResourceChange(ctx context.Context, req *tfplugin6.PlanResourceChange_Request) (*tfplugin6.PlanResourceChange_Response, error) {
	configObj, err := memoryValFromProto(req.Config)
	if err != nil {
		return &tfplugin6.PlanResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to decode configuration value",
				"Configuration value is invalid", err,
			),
		}, nil
	}
	priorObj, err := memoryValFromProto(req.PriorState)
	if err != nil {
		return &tfplugin6.PlanResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to decode prior state value",
				"Prior state value is invalid", err,
			),
		}, nil
	}
	var val cty.Value
	newValue := configObj.GetAttr("new_value")
	if !newValue.IsKnown() && newValue.Range().CouldBeNull() {
		if !p.deferralAllowed {
			// Either using an OpenTofu/Terraform release that lacks
			// deferred actions support or running in a mode where
			// deferred actions are not allowed.
			return &tfplugin6.PlanResourceChange_Response{
				Diagnostics: []*tfplugin6.Diagnostic{
					{
						Severity: tfplugin6.Diagnostic_ERROR,
						Summary:  "New value not yet known",
						Detail:   "The new_value argument's nullness is based on a result that won't be known until the apply phase, and so \"memory\" cannot determine whether you intend to write to it.\n\nTo avoid this problem, either allow deferral for this plan or exclude this resource from the plan until the new value has been finalized.",
						Attribute: &tfplugin6.AttributePath{
							Steps: []*tfplugin6.AttributePath_Step{
								{
									Selector: &tfplugin6.AttributePath_Step_AttributeName{
										AttributeName: "new_value",
									},
								},
							},
						},
					},
				},
			}, nil
		}
		return &tfplugin6.PlanResourceChange_Response{
			PlannedState: req.ProposedNewState,
			Deferred: &tfplugin6.Deferred{
				Reason: tfplugin6.Deferred_RESOURCE_CONFIG_UNKNOWN,
			},
		}, nil
	}
	if priorObj.IsNull() && newValue.IsNull() {
		return &tfplugin6.PlanResourceChange_Response{
			Diagnostics: []*tfplugin6.Diagnostic{
				{
					Severity: tfplugin6.Diagnostic_ERROR,
					Summary:  "New value is required during creation",
					Detail:   "This memory object has not yet been created, so new_value must be set to initialize the memory.",
					Attribute: &tfplugin6.AttributePath{
						Steps: []*tfplugin6.AttributePath_Step{
							{
								Selector: &tfplugin6.AttributePath_Step_AttributeName{
									AttributeName: "new_value",
								},
							},
						},
					},
				},
			},
		}, nil
	}
	if newValue.IsNull() {
		// If there's no new value then we'll just preserve the prior
		// state verbatim, retaining the previous value.
		return &tfplugin6.PlanResourceChange_Response{
			PlannedState: req.PriorState,
		}, nil
	}
	// Because new_value is a write-only attribute, Terraform does not
	// guarantee it will have the same value during the apply phase
	// and so we need to report an unknown value here to allow for
	// the final value to be different.
	val = cty.DynamicVal
	// We'll refer to the configuration again during the apply phase
	// to determine what the final value is.
	newObj := cty.ObjectVal(map[string]cty.Value{
		"new_value": cty.NullVal(newValue.Type()), // read-only attribute, so always null in response
		"value":     val,
	})
	ret, err := memoryValToProto(newObj)
	if err != nil {
		return &tfplugin6.PlanResourceChange_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to serialize planned new state",
				"Could not serialize the planned new state", err,
			),
		}, nil
	}
	return &tfplugin6.PlanResourceChange_Response{
		PlannedState: ret,
	}, nil
}

// ReadResource implements tfplugin6.ProviderServer.
func (p *Provider) ReadResource(ctx context.Context, req *tfplugin6.ReadResource_Request) (*tfplugin6.ReadResource_Response, error) {
	return &tfplugin6.ReadResource_Response{
		NewState: req.CurrentState,
	}, nil
}

// StopProvider implements tfplugin6.ProviderServer.
func (p *Provider) StopProvider(context.Context, *tfplugin6.StopProvider_Request) (*tfplugin6.StopProvider_Response, error) {
	return &tfplugin6.StopProvider_Response{}, nil
}

// UpgradeResourceState implements tfplugin6.ProviderServer.
func (p *Provider) UpgradeResourceState(ctx context.Context, req *tfplugin6.UpgradeResourceState_Request) (*tfplugin6.UpgradeResourceState_Response, error) {
	obj, err := memoryValFromJSON(req.RawState.Json)
	if err != nil {
		return &tfplugin6.UpgradeResourceState_Response{
			Diagnostics: diagnosticsForErr(
				"Failed to upgrade previous run state",
				"Previous run state is invalid", err,
			),
		}, nil
	}
	ret, err := memoryValToProto(obj)
	if err != nil {
		return &tfplugin6.UpgradeResourceState_Response{
			Diagnostics: diagnosticsForErr(
				"Failed reserialize previous run state",
				"Previous run state is invalid", err,
			),
		}, nil
	}
	return &tfplugin6.UpgradeResourceState_Response{
		UpgradedState: ret,
	}, nil
}

// ValidateProviderConfig implements tfplugin6.ProviderServer.
func (p *Provider) ValidateProviderConfig(context.Context, *tfplugin6.ValidateProviderConfig_Request) (*tfplugin6.ValidateProviderConfig_Response, error) {
	return &tfplugin6.ValidateProviderConfig_Response{}, nil
}

// ValidateResourceConfig implements tfplugin6.ProviderServer.
func (p *Provider) ValidateResourceConfig(context.Context, *tfplugin6.ValidateResourceConfig_Request) (*tfplugin6.ValidateResourceConfig_Response, error) {
	return &tfplugin6.ValidateResourceConfig_Response{}, nil
}
