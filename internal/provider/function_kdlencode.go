package provider

import (
	"context"
	"fmt"

	kdl "github.com/calico32/kdl-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

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
		Description: "Encodes a Terraform list of node objects as a KDL document string. " +
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
