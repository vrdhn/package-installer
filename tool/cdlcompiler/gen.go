package main

import (
	"bytes"
	_ "embed"
	"go/format"
	"sort"
	"text/template"
)

type genData struct {
	Pkg         string
	GlobalFlags []*flag
	Topics      []*topic
	Commands    []*command
	AllCommands []*command
	Leafs       []*command
	ParamDefs   []param
	CmdVarName  map[*command]string
	AppName     string
	Tagline     string
}

func generate(cdl *cdlTop, pkgName string) ([]byte, []byte, error) {
	params, err := collectParams(cdl)
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

	helpCmd := &command{
		Name:   "help",
		Desc:   "Show help information",
		Params: map[string]value{"safe": {Kind: "bool", Bool: true}}}
	topCommands := append(append([]*command{}, cdl.Commands...), helpCmd)
	all = append(all, helpCmd)

	for _, c := range all {
		cmdVars[c] = "cmd" + goNameForPath(cmdPath(c))
	}

	data := genData{
		Pkg:         pkgName,
		GlobalFlags: cdl.GlobalFlags,
		Topics:      cdl.Topics,
		Commands:    topCommands,
		AllCommands: all,
		Leafs:       leafs,
		ParamDefs:   params,
		CmdVarName:  cmdVars,
		AppName:     cdl.AppName,
		Tagline:     cdl.Tagline,
	}

	funcs := template.FuncMap{
		"goFieldName":    goFieldName,
		"goTypeForParam": goTypeForParam,
		"goTypeForFlag":  goTypeForFlag,
		"goTypeForArg":   goTypeForArg,
		"lowerFirst":     lowerFirst,
		"goNameForPath":  goNameForPath,
		"cmdPath":        cmdPath,
		"cmdVar": func(c *command) string {
			return cmdVars[c]
		},
		"paramLiteral": func(cmd *command, name string, kind string) string {
			val, _ := resolveParam(cdl, cmd, name)
			return emitParamLiteral(val, kind)
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
