package memory

import (
	"fmt"

	"github.com/apparentlymart/terraform-provider-memory/internal/tfplugin6"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
)

var memoryType = cty.Object(map[string]cty.Type{
	"new_value": cty.DynamicPseudoType,
	"value":     cty.DynamicPseudoType,
})

func memoryValFromJSON(src []byte) (cty.Value, error) {
	return json.Unmarshal(src, memoryType)
}

func memoryValFromMsgpack(src []byte) (cty.Value, error) {
	return msgpack.Unmarshal(src, memoryType)
}

func memoryValFromProto(dv *tfplugin6.DynamicValue) (cty.Value, error) {
	switch {
	case len(dv.Json) != 0:
		return memoryValFromJSON(dv.Json)
	case len(dv.Msgpack) != 0:
		return memoryValFromMsgpack(dv.Msgpack)
	default:
		return cty.NilVal, fmt.Errorf("unsupported dynamic value serialization format")
	}
}

func memoryValToMsgpack(obj cty.Value) ([]byte, error) {
	return msgpack.Marshal(obj, memoryType)
}

func memoryValToProto(obj cty.Value) (*tfplugin6.DynamicValue, error) {
	src, err := memoryValToMsgpack(obj)
	if err != nil {
		return nil, err
	}
	return &tfplugin6.DynamicValue{
		Msgpack: src,
	}, nil
}

func diagnosticForErr(summary, prefix string, err error) *tfplugin6.Diagnostic {
	return &tfplugin6.Diagnostic{
		Severity: tfplugin6.Diagnostic_ERROR,
		Summary:  summary,
		Detail:   fmt.Sprintf("%s: %s.", prefix, err),
	}
}

func diagnosticsForErr(summary, prefix string, err error) []*tfplugin6.Diagnostic {
	if err == nil {
		return nil
	}
	return []*tfplugin6.Diagnostic{
		diagnosticForErr(summary, prefix, err),
	}
}
