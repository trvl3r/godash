package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/spf13/viper"
)

var repo, workdir string

func Fireup() {
	workdir = "work"
	repo = "https://github.expedia.biz/wprevost/tf_hashi_stack"

	LoadConfig()
	CreateWorkDir(workdir)

	err := os.Chdir(workdir)
	if err != nil {
		log.Fatalln(err)
	}

	module := CloneRepo(workdir, repo)
	modulePath := ScanModule(workdir, module)

	Start(modulePath)
}

func Start(module string) {
	var (
		err error
	)

	err = os.Chdir(module)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := os.Stat("killme"); os.IsNotExist(err) {
		fmt.Println("Kicking off new module deployment")
		Deploy(module)
	} else {
		fmt.Println("Kill switch found, destrying the deployment")
		err := os.Remove("killme")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("manual kill switch cleaned up, launching destroy")
		Destroy(module)
	}

}

func Destroy(modPath string) {
	var (
		err     error
		varfile string
	)
	// until we can generate a varfile from input
	varfile = "temp.tfvars"

	err = os.Chdir(modPath)
	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Println("Tf destroy from: ", modPath)
	}

	if _, err := os.Stat("terraform.tfstate"); os.IsNotExist(err) {
		fmt.Println("no statefile found.. exiting")
	} else {
		fmt.Println("state found")
		destroy := exec.Command("terraform", "destroy", "-var-file", varfile, "-input=false", "-auto-approve")
		destroy.Dir = modPath
		var destroy_out bytes.Buffer
		var destroy_err bytes.Buffer
		destroy.Stdout = &destroy_out
		destroy.Stderr = &destroy_err
		err = destroy.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + destroy_err.String())
		}
		fmt.Printf("tf destroy: %q\n", destroy_out.String())
	}
}

func Deploy(modPath string) {
	var (
		err     error
		varfile string
	)

	// until we can generate a varfile from input
	varfile = "temp.tfvars"

	err = os.Chdir(modPath)
	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Println("Tf init from: ", modPath)
	}

	if _, err := os.Stat(".terraform"); os.IsNotExist(err) {
		init := exec.Command("terraform", "init", "-input=false")
		init.Dir = modPath
		var init_out bytes.Buffer
		var init_err bytes.Buffer
		init.Stdout = &init_out
		init.Stderr = &init_err
		err = init.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + init_err.String())
		}
		fmt.Printf("tf init: %q\n", init.String())
	}

	fmt.Printf("Planning...")
	plan := exec.Command("terraform", "plan", "-out=tfplan.tmp", "-var-file", varfile, "-input=false")
	var plan_out bytes.Buffer
	var plan_err bytes.Buffer
	plan.Stdout = &plan_out
	plan.Stderr = &plan_err
	err = plan.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + plan_err.String())
	}
	fmt.Println("Plan generated")
	//fmt.Printf("tf plan: %q\n", plan_out.String())

	show := exec.Command("terraform", "show", "-json", "tfplan.tmp")
	var show_out bytes.Buffer
	var show_err bytes.Buffer
	show.Stdout = &show_out
	err = show.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + show_err.String())
	}

	//fmt.Printf("tf show: %q\n", show_out.String())

	// apply := exec.Command("terraform", "apply", "-input=false", "tfplan")

	fmt.Println("Deployment Started...")
	apply := exec.Command("terraform", "apply", "-input=false", "tfplan.tmp")
	var apply_out bytes.Buffer
	var apply_err bytes.Buffer
	apply.Stdout = &apply_out
	err = apply.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + apply_err.String())
	}
	fmt.Println("Deployment Complete")
	//fmt.Printf("tf apply: %q\n", apply_out.String())
}

func ScanModule(dir string, mod string) string {
	var (
		//cmdOut []byte
		err error
	)

	dir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("working dir is:", dir)

	scanPath := filepath.Join(dir, mod)
	fmt.Println("scanning module at: ", scanPath)
	module, diag := tfconfig.LoadModule(scanPath)
	if diag.Err() != nil {
		log.Fatal(diag.Err())
	}

	fmt.Println("module path is:", module.Path)

	//showModuleMarkdown(module)

	for _, rvars := range module.Variables {
		if rvars.Default == nil {
			fmt.Println("No defaults for: ", rvars.Name)
		}
	}
	return module.Path
}

func CloneRepo(dir string, repo string) string {
	var (
		cmdOut []byte
		err    error
	)

	// err = os.Chdir(dir)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	re := regexp.MustCompile("([^\\/]+$)")
	reponame := re.FindStringSubmatch(repo)

	wd, _ := os.Getwd()
	files, err := ioutil.ReadDir(wd)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if reponame[1] == f.Name() {
			fmt.Println("repo already cloned")
			return f.Name()
		}
	}
	cmdName := "git"
	cmdArgs := []string{"clone", repo}

	if cmdOut, err = exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error cloning the repo: ", err)
		os.Exit(1)
	}
	resp := string(cmdOut)
	fmt.Println("Response:", resp)
	return reponame[1]
}

func CreateWorkDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("Working directory already exists.")
	}
}

type config map[string]interface{}

func LoadConfig() {
	os.Setenv("AWS_PROFILE", "tf_auto")
	viper.SetConfigName("tfstream")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config/")
	// viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var C config

	err = viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	fmt.Printf("Config loaded:\n %v \n", C)
}

func showModuleMarkdown(module *tfconfig.Module) {
	tmpl := template.New("md")
	tmpl.Funcs(template.FuncMap{
		"tt": func(s string) string {
			return "`" + s + "`"
		},
		"commas": func(s []string) string {
			return strings.Join(s, ", ")
		},
		"json": func(v interface{}) (string, error) {
			j, err := json.Marshal(v)
			return string(j), err
		},
		"severity": func(s tfconfig.DiagSeverity) string {
			switch s {
			case tfconfig.DiagError:
				return "Error: "
			case tfconfig.DiagWarning:
				return "Warning: "
			default:
				return ""
			}
		},
	})
	template.Must(tmpl.Parse(markdownTemplate))

	err := tmpl.Execute(os.Stdout, module)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering template: %s\n", err)
		os.Exit(2)
	}
}

const markdownTemplate = `
# Module {{ tt .Path }}

{{- if .RequiredCore}}

Core Version Constraints:
{{- range .RequiredCore }}
* {{ tt . }}
{{- end}}{{end}}

{{- if .Variables}}

## Input Variables
{{- range .Variables }}
* {{ tt .Name }}{{ if .Default }} (default {{ json .Default | tt }}){{else}} (required){{end}}
{{- if .Description}}: {{ .Description }}{{ end }}
{{- end}}{{end}}

{{- if .Diagnostics}}

## Problems
{{- range .Diagnostics }}

## {{ severity .Severity }}{{ .Summary }}{{ if .Pos }}

(at {{ tt .Pos.Filename }} line {{ .Pos.Line }}{{ end }})
{{ if .Detail }}
{{ .Detail }}
{{- end }}

{{- end}}{{end}}
`

const fullMarkdownTemplate = `
# Module {{ tt .Path }}

{{- if .RequiredCore}}

Core Version Constraints:
{{- range .RequiredCore }}
* {{ tt . }}
{{- end}}{{end}}

{{- if .Variables}}

## Input Variables
{{- range .Variables }}
* {{ tt .Name }}{{ if .Default }} (default {{ json .Default | tt }}){{else}} (required){{end}}
{{- if .Description}}: {{ .Description }}{{ end }}
{{- end}}{{end}}

{{- if .Outputs}}

## Output Values
{{- range .Outputs }}
* {{ tt .Name }}{{ if .Description}}: {{ .Description }}{{ end }}
{{- end}}{{end}}

{{- if .ManagedResources}}

## Managed Resources
{{- range .ManagedResources }}
* {{ printf "%s.%s" .Type .Name | tt }} from {{ tt .Provider.Name }}
{{- end}}{{end}}

{{- if .DataResources}}

## Data Resources
{{- range .DataResources }}
* {{ printf "data.%s.%s" .Type .Name | tt }} from {{ tt .Provider.Name }}
{{- end}}{{end}}

{{- if .ModuleCalls}}

## Child Modules
{{- range .ModuleCalls }}
* {{ tt .Name }} from {{ tt .Source }}{{ if .Version }} ({{ tt .Version }}){{ end }}
{{- end}}{{end}}

{{- if .Diagnostics}}

## Problems
{{- range .Diagnostics }}

## {{ severity .Severity }}{{ .Summary }}{{ if .Pos }}

(at {{ tt .Pos.Filename }} line {{ .Pos.Line }}{{ end }})
{{ if .Detail }}
{{ .Detail }}
{{- end }}

{{- end}}{{end}}
`
