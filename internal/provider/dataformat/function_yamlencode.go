package dataformat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"gopkg.in/yaml.v3"
)

var _ function.Function = (*YAMLEncodeFunction)(nil)

type YAMLEncodeFunction struct{}

func NewYAMLEncodeFunction() function.Function {
	return &YAMLEncodeFunction{}
}

func (f *YAMLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "yamlencode"
}

func (f *YAMLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as YAML with full formatting control",
		MarkdownDescription: "Encodes a Terraform value as a YAML string. Unlike Terraform's built-in `yamlencode`, this " +
			"defaults to block style, uses literal block scalars (`|`) for multi-line strings, and supports inline comments " +
			"through an `options.comments` map.\n\n" +
			"Pass an optional `options` object with these keys:\n\n" +
			"- `indent` (number): indent width in spaces (default `2`).\n" +
			"- `style` (string): `\"block\"` (default) or `\"flow\"`.\n" +
			"- `quote_style` (string): `\"double\"`, `\"single\"`, or `\"plain\"`.\n" +
			"- `null` (string): how to render nulls (default empty).\n" +
			"- `sort_keys` (bool): sort object keys (default `true`).\n" +
			"- `comments` (object): mirrored structure with string values that become `# ` comments before the matching key.\n\n" +
			"**Common uses:** generating Kubernetes manifests, GitHub Actions workflows, Helm values files, or any other YAML " +
			"configuration that gets reviewed and edited by humans.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as YAML.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name: "options",
			Description: "An optional options object. Supported keys: " +
				"\"indent\" (number, default 2), " +
				"\"flow_level\" (number, default 0: all block; -1: all flow), " +
				"\"multiline\" (string: \"literal\", \"folded\", or \"quoted\"), " +
				"\"quote_style\" (string: \"auto\", \"double\", or \"single\"), " +
				"\"null_value\" (string: \"null\", \"~\", or \"\"), " +
				"\"sort_keys\" (bool, default true), " +
				"\"comments\" (object: mirrored structure for # comments). " +
				"Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

type yamlEncodeOpts struct {
	indent     int
	flowLevel  int
	multiline  string // "literal", "folded", "quoted"
	quoteStyle string // "auto", "double", "single"
	nullValue  string // "null", "~", ""
	sortKeys   bool
	dedupe     bool
	comments   attr.Value
}

func defaultYAMLOpts() yamlEncodeOpts {
	return yamlEncodeOpts{
		indent:     2,
		flowLevel:  0,
		multiline:  "literal",
		quoteStyle: "auto",
		nullValue:  "null",
		sortKeys:   true,
		dedupe:     false,
	}
}

func (f *YAMLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	opts := defaultYAMLOpts()

	if len(optsArgs) == 1 {
		var err error
		opts, err = parseYAMLEncodeOpts(optsArgs[0])
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	node := goToYAMLNode(goVal, opts, 0)

	// Deduplicate identical subtrees with anchors/aliases.
	if opts.dedupe {
		dedupeYAMLNodes(node)
	}

	// Apply comments.
	if opts.comments != nil {
		applyYAMLComments(node, opts.comments)
	}

	// Marshal via yaml.v3 encoder for indent control.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(opts.indent)
	if err := enc.Encode(node); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode YAML: "+err.Error()))
		return
	}
	enc.Close()

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, buf.String()))
}

func parseYAMLEncodeOpts(optsArg types.Dynamic) (yamlEncodeOpts, error) {
	opts := defaultYAMLOpts()

	obj, ok := optsArg.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		return opts, fmt.Errorf("options must be an object, got %T", optsArg.UnderlyingValue())
	}

	attrs := obj.Attributes()

	if v, ok := attrs["indent"]; ok {
		nv, ok := v.(basetypes.NumberValue)
		if !ok {
			return opts, fmt.Errorf("\"indent\" must be a number")
		}
		f, _ := nv.ValueBigFloat().Int64()
		opts.indent = int(f)
	}

	if v, ok := attrs["flow_level"]; ok {
		nv, ok := v.(basetypes.NumberValue)
		if !ok {
			return opts, fmt.Errorf("\"flow_level\" must be a number")
		}
		f, _ := nv.ValueBigFloat().Int64()
		opts.flowLevel = int(f)
	}

	if s, err := getStringOption(attrs, "multiline"); err != nil {
		return opts, err
	} else if s != "" {
		switch s {
		case "literal", "folded", "quoted":
			opts.multiline = s
		default:
			return opts, fmt.Errorf("\"multiline\" must be \"literal\", \"folded\", or \"quoted\", got %q", s)
		}
	}

	if s, err := getStringOption(attrs, "quote_style"); err != nil {
		return opts, err
	} else if s != "" {
		switch s {
		case "auto", "double", "single":
			opts.quoteStyle = s
		default:
			return opts, fmt.Errorf("\"quote_style\" must be \"auto\", \"double\", or \"single\", got %q", s)
		}
	}

	if s, err := getStringOption(attrs, "null_value"); err != nil {
		return opts, err
	} else if s != "" || attrs["null_value"] != nil {
		// Allow empty string as a valid null representation.
		if s == "" {
			opts.nullValue = ""
		} else {
			switch s {
			case "null", "~":
				opts.nullValue = s
			default:
				return opts, fmt.Errorf("\"null_value\" must be \"null\", \"~\", or \"\", got %q", s)
			}
		}
	}

	if v, ok := attrs["sort_keys"]; ok {
		bv, ok := v.(basetypes.BoolValue)
		if !ok {
			return opts, fmt.Errorf("\"sort_keys\" must be a bool")
		}
		opts.sortKeys = bv.ValueBool()
	}

	if v, ok := attrs["dedupe"]; ok {
		bv, ok := v.(basetypes.BoolValue)
		if !ok {
			return opts, fmt.Errorf("\"dedupe\" must be a bool")
		}
		opts.dedupe = bv.ValueBool()
	}

	if c, ok := attrs["comments"]; ok {
		opts.comments = c
	}

	return opts, nil
}

// goToYAMLNode converts a Go value to a yaml.Node tree with style control.
func goToYAMLNode(v interface{}, opts yamlEncodeOpts, depth int) *yaml.Node {
	useFlow := opts.flowLevel < 0 || (opts.flowLevel > 0 && depth >= opts.flowLevel)

	switch val := v.(type) {
	case nil:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: opts.nullValue,
		}

	case bool:
		s := "false"
		if val {
			s = "true"
		}
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!bool",
			Value: s,
		}

	case string:
		return yamlStringNode(val, opts)

	case int64:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!int",
			Value: strconv.FormatInt(val, 10),
		}

	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) {
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
				Value: strconv.FormatInt(int64(val), 10),
			}
		}
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!float",
			Value: strconv.FormatFloat(val, 'f', -1, 64),
		}

	case []interface{}:
		node := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
		}
		if useFlow {
			node.Style = yaml.FlowStyle
		}
		for _, item := range val {
			node.Content = append(node.Content, goToYAMLNode(item, opts, depth+1))
		}
		return node

	case map[string]interface{}:
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		if useFlow {
			node.Style = yaml.FlowStyle
		}

		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		if opts.sortKeys {
			sort.Strings(keys)
		}

		for _, k := range keys {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: k,
			}
			valNode := goToYAMLNode(val[k], opts, depth+1)
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node

	case orderedMap:
		// From goValueForJSONEncode — preserve order.
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		if useFlow {
			node.Style = yaml.FlowStyle
		}
		for _, entry := range val {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: entry.Key,
			}
			valNode := goToYAMLNode(entry.Value, opts, depth+1)
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node

	default:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%v", v),
		}
	}
}

// yamlStringNode creates a scalar node for a string with appropriate style.
func yamlStringNode(s string, opts yamlEncodeOpts) *yaml.Node {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: s,
	}

	// Apply quoting style.
	switch opts.quoteStyle {
	case "double":
		node.Style = yaml.DoubleQuotedStyle
	case "single":
		node.Style = yaml.SingleQuotedStyle
	default:
		// "auto" — yaml.v3 quotes only when needed.
	}

	// Multi-line strings get special treatment (overrides quote_style).
	if strings.Contains(s, "\n") {
		switch opts.multiline {
		case "literal":
			node.Style = yaml.LiteralStyle
		case "folded":
			node.Style = yaml.FoldedStyle
		case "quoted":
			node.Style = yaml.DoubleQuotedStyle
		}
	}

	return node
}

// applyYAMLComments walks the YAML node tree and the comments value in parallel,
// setting HeadComment on matching keys.
func applyYAMLComments(node *yaml.Node, comments attr.Value) {
	commentsObj, ok := comments.(basetypes.ObjectValue)
	if !ok {
		return
	}

	applyYAMLCommentsToNode(node, commentsObj.Attributes())
}

func applyYAMLCommentsToNode(node *yaml.Node, commentsMap map[string]attr.Value) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		// Content is alternating key/value nodes.
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			commentVal, ok := commentsMap[keyNode.Value]
			if !ok {
				continue
			}

			switch cv := commentVal.(type) {
			case basetypes.StringValue:
				keyNode.HeadComment = cv.ValueString()
			case basetypes.ObjectValue:
				applyYAMLCommentsToNode(valNode, cv.Attributes())
			}
		}

	case yaml.SequenceNode:
		for key, commentVal := range commentsMap {
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(node.Content) {
				continue
			}

			switch cv := commentVal.(type) {
			case basetypes.StringValue:
				node.Content[idx].HeadComment = cv.ValueString()
			case basetypes.ObjectValue:
				applyYAMLCommentsToNode(node.Content[idx], cv.Attributes())
			}
		}
	}
}

// dedupeYAMLNodes finds identical non-scalar subtrees in the node tree,
// anchors the first occurrence, and replaces subsequent ones with aliases.
func dedupeYAMLNodes(root *yaml.Node) {
	// Phase 1: Hash every node and collect pointers by hash.
	hashes := map[string][]*yaml.Node{}
	hashNode(root, hashes)

	// Phase 2: Filter to hashes with 2+ occurrences of non-trivial nodes.
	anchorCounter := 0
	anchored := map[string]string{} // hash → anchor name

	for hash, nodes := range hashes {
		if len(nodes) < 2 {
			continue
		}
		// Skip scalar nodes — not worth anchoring individual strings/numbers.
		if nodes[0].Kind == yaml.ScalarNode {
			continue
		}
		// Skip very small nodes (fewer than 2 content items).
		if len(nodes[0].Content) < 2 {
			continue
		}

		anchorCounter++
		name := fmt.Sprintf("_ref%d", anchorCounter)
		anchored[hash] = name
		nodes[0].Anchor = name
	}

	if len(anchored) == 0 {
		return
	}

	// Phase 3: Walk again and replace duplicate nodes with alias nodes.
	replaceWithAliases(root, anchored, hashes)
}

// hashNode computes a content hash for a yaml.Node and all its children,
// registering each non-scalar node in the hashes map.
func hashNode(node *yaml.Node, hashes map[string][]*yaml.Node) string {
	if node == nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d:%s:%s:", node.Kind, node.Tag, node.Value)))

	for _, child := range node.Content {
		childHash := hashNode(child, hashes)
		h.Write([]byte(childHash))
	}

	hash := hex.EncodeToString(h.Sum(nil))

	// Only track non-scalar nodes for deduplication.
	if node.Kind != yaml.ScalarNode {
		hashes[hash] = append(hashes[hash], node)
	}

	return hash
}

// replaceWithAliases walks the tree and replaces content nodes that match
// an anchored hash (but aren't the anchor themselves) with alias nodes.
func replaceWithAliases(node *yaml.Node, anchored map[string]string, hashes map[string][]*yaml.Node) {
	if node == nil {
		return
	}

	for i, child := range node.Content {
		if child == nil || child.Kind == yaml.ScalarNode {
			continue
		}

		childHash := computeHash(child)
		anchorName, isAnchored := anchored[childHash]

		if isAnchored && child.Anchor == "" {
			// This is a duplicate — replace with alias.
			// Find the anchored node to point to.
			firstNode := hashes[childHash][0]
			node.Content[i] = &yaml.Node{
				Kind:  yaml.AliasNode,
				Alias: firstNode,
				Value: anchorName,
			}
		} else {
			replaceWithAliases(child, anchored, hashes)
		}
	}
}

// computeHash computes the content hash for a single node (without registering).
func computeHash(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d:%s:%s:", node.Kind, node.Tag, node.Value)))
	for _, child := range node.Content {
		h.Write([]byte(computeHash(child)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
