package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"

	rice "github.com/GeertJohan/go.rice"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/mikemackintosh/bakery/cli"
	"github.com/mikemackintosh/bakery/config"
	"github.com/mikemackintosh/bakery/pantry"
	"github.com/zclconf/go-cty/cty"
)

var runList = &Runlist{
	Items: map[string]pantry.PantryInterface{},
	// CompletedItems: map[string]pantry.PantryInterface{},
}

// Variable contains variables
type Variable struct {
	Name    string         `hcl:"name,label"`
	Default hcl.Attributes `hcl:"default,remain"`
}

// Bakery is the parent struct
type Bakery struct {
	Variables []*Variable     `hcl:"variable,block"`
	Dmgs      []*pantry.Dmg   `hcl:"dmg,block"`
	Pkgs      []*pantry.Pkg   `hcl:"pkg,block"`
	Shells    []*pantry.Shell `hcl:"shell,block"`
	Zips      []*pantry.Zip   `hcl:"zip,block"`
	Gits      []*pantry.Git   `hcl:"git,block"`
	Brews     []*pantry.Brew  `hcl:"brew,block"`
	Fonts     []*pantry.Font  `hcl:"font,block"`
}

// Runlist contains a list of items
type Runlist struct {
	Items map[string]pantry.PantryInterface
}

// Add adds an item to the Item list
func (rl *Runlist) Add(name string, pi pantry.PantryInterface) {
	rl.Items[name] = pi
}

func main() {
	flag.Parse()
	var err error
	var file *hcl.File
	var diags hcl.Diagnostics
	var p = hclparse.NewParser()

	// Override the config registry
	config.Registry.TempDir = cli.FlagTempDir

	if cli.FlagBundle {
		// find a rice.Box
		templateBox, err := rice.FindBox("recipes")
		if err != nil {
			log.Fatal(err)
		}

		// get file contents as string
		templateString, err := templateBox.String("config.yum")
		if err != nil {
			log.Fatal(err)
		}

		file, diags = p.ParseHCL([]byte(templateString), "config.yum")
	} else {
		file, diags = p.ParseHCLFile(cli.FlagRecipe)
	}

	if len(diags) != 0 {
		for _, diag := range diags {
			fmt.Printf("- %s\n", diag)
		}
		return
	}

	// Make the temp file directory
	// TODO: refactor this out
	err = os.MkdirAll(cli.FlagTempDir, 0755)
	if err != nil {
		cli.ErrorAndExit(err)
	}

	body := file.Body

	var bakery Bakery
	diags = gohcl.DecodeBody(body, nil, &bakery)
	if len(diags) != 0 {
		for _, diag := range diags {
			fmt.Printf("decoding - %s\n", diag)
		}
		return
	}

	variables := map[string]cty.Value{}
	for _, v := range bakery.Variables {
		if len(v.Default) == 0 {
			continue
		}

		val, diags := v.Default["default"].Expr.Value(nil)
		if len(diags) != 0 {
			for _, diag := range diags {
				fmt.Printf("decoding - %s\n", diag)
			}
			return
		}

		variables[v.Name] = val
	}

	evalContext := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(nil),
		},
	}

	rootVal := reflect.ValueOf(bakery)
	for i := 0; i < rootVal.NumField(); i++ {
		for sliceinc := 0; sliceinc < rootVal.Field(i).Len(); sliceinc++ {
			entry := rootVal.Field(i).Index(sliceinc)
			params := []reflect.Value{reflect.ValueOf(evalContext)}
			r := entry.MethodByName("Parse").Call(params)
			if len(r) > 0 && r[0].Interface() != nil {
				cli.ErrorAndExit(fmt.Errorf("%v", r[0].Interface()))
			}

			runList.Add(entry.Elem().FieldByName("Name").String(), entry.Interface().(pantry.PantryInterface))
		}
	}

	for name, module := range runList.Items {
		err := RunItem(name, module)
		if err != nil {
			panic(err)
		}
	}
}

// RunItem will perform dependency validation and perform the action
func RunItem(name string, module pantry.PantryInterface) error {
	if !module.Ready() {
		deps := module.GetDependencies()
		if len(deps) > 0 {
			for _, k := range deps {
				if !runList.Items[k].Ready() {
					cli.Debug(cli.INFO, fmt.Sprintf("%s has an Unmet dependency: %s, running now\n", name, k), nil)
					RunItem(k, runList.Items[k])
				}
			}
		}

		cli.Debug(cli.INFO, "Baking", name)
		m := module.(pantry.PantryInterface)

		if m.ValidateOnlyIf() {
			return nil
		}

		if m.ValidateNotIf() {
			return nil
		}

		m.Bake()
		module.Baked()
	}
	return nil
}
