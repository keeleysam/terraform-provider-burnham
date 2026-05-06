package dataformat

import (
	"context"
	"fmt"
	"strings"

	kdl "github.com/calico32/kdl-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
		MarkdownDescription: "Decodes a KDL document string into a Terraform list of node objects. " +
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

var _ function.Function = (*KDLEncodeFunction)(nil)

type KDLEncodeFunction struct{}

func NewKDLEncodeFunction() function.Function {
	return &KDLEncodeFunction{}
}

func (f *KDLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "kdlencode"
}

func (f *KDLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as a KDL document",
		MarkdownDescription: "Encodes a Terraform list of node objects as a KDL document string. " +
			"Each node object should have \"name\" (string), \"args\" (list), \"props\" (map), " +
			"and \"children\" (list of child nodes). Default output is KDL v2; " +
			"pass an options object with version=\"v1\" for KDL v1 output.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "A list of node objects to encode as KDL.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name: "options",
			Description: "An optional options object. Supported keys: " +
				"\"version\" (string) — \"v2\" (default) or \"v1\". " +
				"Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *KDLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	version := kdl.Version2
	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue())))
			return
		}
		vStr, err := getStringOption(obj.Attributes(), "version")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		switch vStr {
		case "v1":
			version = kdl.Version1
		case "v2", "":
			version = kdl.Version2
		default:
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("\"version\" must be \"v1\" or \"v2\", got %q", vStr)))
			return
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	doc, err := terraformToKDLDoc(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert to KDL: "+err.Error()))
		return
	}

	result, err := kdl.EmitToString(doc, kdl.WithVersion(version))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to emit KDL: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

// terraformToKDLDoc converts a Terraform tuple/list of node objects to a KDL Document.
func terraformToKDLDoc(v attr.Value) (*kdl.Document, error) {
	var elements []attr.Value

	switch val := v.(type) {
	case basetypes.TupleValue:
		elements = val.Elements()
	case basetypes.ListValue:
		elements = val.Elements()
	default:
		return nil, fmt.Errorf("value must be a list of node objects, got %T", v)
	}

	doc := kdl.NewDocument()
	for i, elem := range elements {
		node, err := terraformToKDLNode(elem)
		if err != nil {
			return nil, fmt.Errorf("node %d: %w", i, err)
		}
		doc.AddNode(node)
	}

	return doc, nil
}

// terraformToKDLNode converts a Terraform object to a KDL Node.
func terraformToKDLNode(v attr.Value) (*kdl.Node, error) {
	obj, ok := v.(basetypes.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("node must be an object, got %T", v)
	}

	attrs := obj.Attributes()

	// Name (required)
	nameAttr, ok := attrs["name"]
	if !ok {
		return nil, fmt.Errorf("node must have a \"name\" key")
	}
	nameStr, ok := nameAttr.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("\"name\" must be a string")
	}
	node := kdl.NewNode(nameStr.ValueString())

	// Args (optional)
	if argsAttr, ok := attrs["args"]; ok {
		var argElems []attr.Value
		switch av := argsAttr.(type) {
		case basetypes.TupleValue:
			argElems = av.Elements()
		case basetypes.ListValue:
			argElems = av.Elements()
		}
		for _, arg := range argElems {
			kv, err := terraformToKDLValue(arg)
			if err != nil {
				return nil, err
			}
			node.AddArgument(kv)
		}
	}

	// Props (optional)
	if propsAttr, ok := attrs["props"]; ok {
		if propsObj, ok := propsAttr.(basetypes.ObjectValue); ok {
			for key, val := range propsObj.Attributes() {
				kv, err := terraformToKDLValue(val)
				if err != nil {
					return nil, err
				}
				node.AddProperty(key, kv)
			}
		}
	}

	// Children (optional)
	if childrenAttr, ok := attrs["children"]; ok {
		childDoc, err := terraformToKDLDoc(childrenAttr)
		if err != nil {
			return nil, fmt.Errorf("children: %w", err)
		}
		for _, child := range childDoc.Nodes {
			node.AddChild(child)
		}
	}

	return node, nil
}

// terraformToKDLValue converts a Terraform attr.Value to a KDL Value.
func terraformToKDLValue(v attr.Value) (kdl.Value, error) {
	if v.IsNull() {
		return kdl.NewNull(), nil
	}

	switch val := v.(type) {
	case basetypes.StringValue:
		return kdl.NewString(val.ValueString()), nil
	case basetypes.BoolValue:
		return kdl.NewBool(val.ValueBool()), nil
	case basetypes.NumberValue:
		f := val.ValueBigFloat()
		if f.IsInt() {
			i, _ := f.Int64()
			return kdl.NewInt(int(i)), nil
		}
		fv, _ := f.Float64()
		return kdl.NewFloat(fv), nil
	default:
		return kdl.NewString(fmt.Sprintf("%v", v)), nil
	}
}
