package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"sort"
	"text/template"
)

type genData struct {
	Pkg         string
	GlobalFlags []flag
	Topics      []topic
	Commands    []*command
	AllCommands []*command
	Leafs       []*command
	AttrDefs    []attr
	CmdVarName  map[*command]string
	AppName     string
	Tagline     string
}

func generate(cdl *cdlTop, pkgName string) ([]byte, []byte, error) {
	attrs, err := collectAttrs(cdl)
	if err != nil {
		return nil, nil, err
	}

	leafs := collectLeafCommands(cdl.Commands)
	sort.Slice(leafs, func(i, j int) bool { return cmdPath(leafs[i]) < cmdPath(leafs[j]) })

	cmdVars := map[*command]string{}
	var all []*command
	walkCommands(cdl.Commands, func(c *command) {
		all = append(all, c)
	})

	allGlobalFlags := append([]flag{
		{Name: "help", Short: "h", Type: "bool", Desc: "Show help information"},
	}, cdl.GlobalFlags...)

	for _, c := range all {
		cmdVars[c] = "cmd" + goNameForPath(cmdPath(c))
	}

	data := genData{
		Pkg:         pkgName,
		GlobalFlags: allGlobalFlags,
		Topics:      cdl.Topics,
		Commands:    cdl.Commands,
		AllCommands: all,
		Leafs:       leafs,
		AttrDefs:    attrs,
		CmdVarName:  cmdVars,
		AppName:     cdl.AppName,
		Tagline:     cdl.Tagline,
	}

	funcs := template.FuncMap{
		"goFieldName":   goFieldName,
		"goTypeForAttr": goTypeForAttr,
		"goTypeForFlag": goTypeForFlag,
		"goTypeForArg":  goTypeForArg,
		"lowerFirst":    lowerFirst,
		"goNameForPath": goNameForPath,
		"cmdPath":       cmdPath,
		"cmdVar": func(c *command) string {
			return cmdVars[c]
		},
		"attrLiteral": func(cmd *command, name string, kind string) string {
			val, _ := resolveAttr(cdl, cmd, name)
			return emitAttrLiteral(val, kind)
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	tmpl, err := template.New("cli").Funcs(funcs).Parse(sourceTemplate)
	if err != nil {
		return nil, nil, err
	}

	sup, err := template.New("cli").Funcs(funcs).Parse(supportTemplate)
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, nil, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, nil, err
	}

	var buf1 bytes.Buffer
	if err := sup.Execute(&buf1, data); err != nil {
		return nil, nil, err
	}

	support, err := format.Source(buf1.Bytes())
	if err != nil {
		return nil, nil, err
	}

	return formatted, support, nil
}

//go:embed source.tmpl
var sourceTemplate string

//go:embed support.tmpl
var supportTemplate string
