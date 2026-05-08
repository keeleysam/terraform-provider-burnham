package dataformat

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*CSVEncodeFunction)(nil)

type CSVEncodeFunction struct{}

func NewCSVEncodeFunction() function.Function {
	return &CSVEncodeFunction{}
}

func (f *CSVEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "csvencode"
}

func (f *CSVEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a list of objects as a CSV string",
		MarkdownDescription: "Encodes a list of objects as a CSV string. Each object becomes a row; object keys become columns.\n\nBy default columns are sorted alphabetically and a header row is written. Pass an optional `options` object:\n\n- `columns` (list of strings): explicit column ordering — only listed columns are included.\n- `no_header` (bool): omit the header row.\n\nAll cell values are converted to strings: numbers render as their string representation, bools as `\"true\"`/`\"false\"`, and nulls as empty fields. Nested values (lists, objects) are not supported and produce an error.\n\n**Common uses:** generating CSV inputs for downstream loaders, exporting lookup tables, or producing reproducible spreadsheet-friendly output. Terraform has a built-in `csvdecode` for the reverse direction.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "rows",
				Description: "A list of objects to encode as CSV rows.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name: "options",
			Description: `An optional options object. Supported keys: ` +
				`"columns" (list of strings) — column names in the desired order; ` +
				`"no_header" (bool) — if true, omit the header row. ` +
				`Pass at most one options object.`,
		},
		Return: function.StringReturn{},
	}
}

func (f *CSVEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var rowsVal types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &rowsVal, &optsArgs))
	if resp.Error != nil {
		return
	}

	if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	// Parse options.
	var columns []string
	var noHeader bool
	if len(optsArgs) == 1 {
		var err error
		columns, noHeader, err = parseCSVEncodeOptions(optsArgs[0])
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
	}

	// Extract rows from the dynamic value.
	rows, err := extractCSVRows(rowsVal.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	if len(rows) == 0 {
		if noHeader || columns == nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, ""))
			return
		}
		// Just headers, no data rows.
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.Write(columns)
		w.Flush()
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, buf.String()))
		return
	}

	// Auto-detect columns if not specified.
	if columns == nil {
		colSet := map[string]bool{}
		for _, row := range rows {
			for k := range row {
				colSet[k] = true
			}
		}
		columns = make([]string, 0, len(colSet))
		for k := range colSet {
			columns = append(columns, k)
		}
		sort.Strings(columns)
	}

	// Write CSV.
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if !noHeader {
		w.Write(columns)
	}

	for i, row := range rows {
		record := make([]string, len(columns))
		for j, col := range columns {
			val, ok := row[col]
			if !ok {
				record[j] = ""
				continue
			}
			s, err := csvCellToString(val)
			if err != nil {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Row %d, column %q: %s", i, col, err.Error())))
				return
			}
			record[j] = s
		}
		w.Write(record)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("CSV write error: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, buf.String()))
}

// parseCSVEncodeOptions extracts columns and no_header from the options dynamic value.
func parseCSVEncodeOptions(opts types.Dynamic) ([]string, bool, error) {
	obj, ok := opts.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		return nil, false, fmt.Errorf("options must be an object, got %T", opts.UnderlyingValue())
	}

	attrs := obj.Attributes()

	var columns []string
	var noHeader bool

	if colVal, ok := attrs["columns"]; ok {
		switch cv := colVal.(type) {
		case basetypes.TupleValue:
			for i, elem := range cv.Elements() {
				sv, ok := elem.(basetypes.StringValue)
				if !ok {
					return nil, false, fmt.Errorf("columns[%d] must be a string", i)
				}
				columns = append(columns, sv.ValueString())
			}
		case basetypes.ListValue:
			for i, elem := range cv.Elements() {
				sv, ok := elem.(basetypes.StringValue)
				if !ok {
					return nil, false, fmt.Errorf("columns[%d] must be a string", i)
				}
				columns = append(columns, sv.ValueString())
			}
		default:
			return nil, false, fmt.Errorf("columns must be a list of strings, got %T", colVal)
		}
	}

	if nhVal, ok := attrs["no_header"]; ok {
		bv, ok := nhVal.(basetypes.BoolValue)
		if !ok {
			return nil, false, fmt.Errorf("no_header must be a bool, got %T", nhVal)
		}
		noHeader = bv.ValueBool()
	}

	return columns, noHeader, nil
}

// extractCSVRows converts a Terraform list/tuple of objects into a slice of
// string-keyed maps with raw attr.Value values.
func extractCSVRows(v attr.Value) ([]map[string]attr.Value, error) {
	var elements []attr.Value

	switch val := v.(type) {
	case basetypes.TupleValue:
		elements = val.Elements()
	case basetypes.ListValue:
		elements = val.Elements()
	default:
		return nil, fmt.Errorf("rows must be a list of objects, got %T", v)
	}

	rows := make([]map[string]attr.Value, len(elements))
	for i, elem := range elements {
		obj, ok := elem.(basetypes.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("row %d must be an object, got %T", i, elem)
		}
		rows[i] = obj.Attributes()
	}

	return rows, nil
}

// csvCellToString converts a Terraform attr.Value to a string for CSV output.
func csvCellToString(v attr.Value) (string, error) {
	if v.IsNull() || v.IsUnknown() {
		return "", nil
	}

	switch val := v.(type) {
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		f := val.ValueBigFloat()
		return f.Text('f', -1), nil
	case basetypes.BoolValue:
		if val.ValueBool() {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unsupported type %T (only strings, numbers, bools, and nulls are supported)", v)
	}
}
