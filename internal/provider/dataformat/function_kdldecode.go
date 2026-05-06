package dataformat

import (
	"context"
	"fmt"
	"strings"

	kdl "github.com/calico32/kdl-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*KDLDecodeFunction)(nil)

type KDLDecodeFunction struct{}

func NewKDLDecodeFunction() function.Function {
	return &KDLDecodeFunction{}
}

func (f *KDLDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "kdldecode"
}

func (f *KDLDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a KDL document into a Terraform value",
		Description: "Decodes a KDL document string into a Terraform list of node objects. " +
			"Each node has \"name\" (string), \"args\" (list of values), \"props\" (map of values), " +
			"and \"children\" (list of child nodes). Supports both KDL v1 and v2 input.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A KDL document string to parse.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *KDLDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	doc, err := kdl.Parse(strings.NewReader(input))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse KDL: "+err.Error()))
		return
	}

	tfVal, err := kdlDocToTerraform(doc)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert KDL: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// kdlDocToTerraform converts a KDL document to a Terraform tuple of node objects.
func kdlDocToTerraform(doc *kdl.Document) (attr.Value, error) {
	nodes := doc.Nodes
	if len(nodes) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
	}

	elemTypes := make([]attr.Type, len(nodes))
	elemValues := make([]attr.Value, len(nodes))

	for i, node := range nodes {
		val, err := kdlNodeToTerraform(node)
		if err != nil {
			return nil, fmt.Errorf("node %d (%q): %w", i, node.Name(), err)
		}
		elemTypes[i] = val.Type(nil)
		elemValues[i] = val
	}

	return types.TupleValueMust(elemTypes, elemValues), nil
}

// kdlNodeToTerraform converts a single KDL node to a Terraform object.
func kdlNodeToTerraform(node *kdl.Node) (attr.Value, error) {
	attrTypes := map[string]attr.Type{}
	attrValues := map[string]attr.Value{}

	// Name
	attrTypes["name"] = types.StringType
	attrValues["name"] = types.StringValue(node.Name())

	// Args
	args := node.Arguments()
	argTypes := make([]attr.Type, len(args))
	argValues := make([]attr.Value, len(args))
	for i, arg := range args {
		v, err := kdlValueToTerraform(arg)
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		argTypes[i] = v.Type(nil)
		argValues[i] = v
	}
	argsTuple := types.TupleValueMust(argTypes, argValues)
	attrTypes["args"] = argsTuple.Type(nil)
	attrValues["args"] = argsTuple

	// Props
	props := node.Properties()
	propOrder := node.PropertyOrder()
	propTypes := make(map[string]attr.Type, len(props))
	propValues := make(map[string]attr.Value, len(props))
	for _, key := range propOrder {
		val := props[key]
		v, err := kdlValueToTerraform(val)
		if err != nil {
			return nil, fmt.Errorf("prop %q: %w", key, err)
		}
		propTypes[key] = v.Type(nil)
		propValues[key] = v
	}
	propsObj := types.ObjectValueMust(propTypes, propValues)
	attrTypes["props"] = propsObj.Type(nil)
	attrValues["props"] = propsObj

	// Children
	children := node.Children()
	childrenVal, err := kdlDocToTerraform(children)
	if err != nil {
		return nil, fmt.Errorf("children: %w", err)
	}
	attrTypes["children"] = childrenVal.Type(nil)
	attrValues["children"] = childrenVal

	return types.ObjectValueMust(attrTypes, attrValues), nil
}

// kdlValueToTerraform converts a KDL value to a Terraform value.
func kdlValueToTerraform(v kdl.Value) (attr.Value, error) {
	switch v.Kind() {
	case kdl.String:
		return types.StringValue(v.RawValue().(string)), nil
	case kdl.Int:
		return goToTerraformValue(v.RawValue())
	case kdl.Float:
		return goToTerraformValue(v.RawValue())
	case kdl.Bool:
		return types.BoolValue(v.RawValue().(bool)), nil
	case kdl.Null:
		return types.StringNull(), nil
	case kdl.BigInt:
		return goToTerraformValue(v.RawValue())
	case kdl.BigFloat:
		return goToTerraformValue(v.RawValue())
	default:
		return types.StringValue(fmt.Sprintf("%v", v.RawValue())), nil
	}
}
