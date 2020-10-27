package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var TempDirBase string

const TempDirPattern = "*"

type program struct {
	data          *schema.ResourceData
	context       context.Context
	name          string
	currentTmpDir string
	inputTmpDir   string
	files         map[string]string
	perms         map[string]os.FileMode
}

func init() {
	var dir string
	if envDir, ok := os.LookupEnv("TF_DATA_DIR"); ok {
		dir = envDir
	} else {
		dir = ".terraform"
	}
	TempDirBase = fmt.Sprintf("%s/terraform-provider-external", dir)
}

func Program(ctx context.Context, data *schema.ResourceData) *program {
	oldStateV, newStateV := data.GetChange("state")
	p := &program{
		context: ctx,
		data:    data,
	}
	p.inputTmpDir = data.Get("program_tmp_dir").(string)

	p.files = map[string]string{
		"input":            p.data.Get("input").(string),
		"input_sensitive":  p.data.Get("input_sensitive").(string),
		"output":           p.data.Get("output").(string),
		"output_sensitive": p.data.Get("output_sensitive").(string),
		"state":            newStateV.(string),
		"old_state":        oldStateV.(string),
		"id":               p.data.Id(),
	}

	p.perms = map[string]os.FileMode{
		"output":           0200,
		"output_sensitive": 0200,
		"state":            0600,
		"id":               0600,
	}

	return p
}

func (p *program) openDir() (diags diag.Diagnostics) {
	cwd, err := os.Getwd()
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error retrieving current working director"),
			Detail:   err.Error(),
		})
		return
	}
	if p.inputTmpDir != "" {
		if err := os.MkdirAll(p.inputTmpDir, 0700); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error creating temporary directory parent %s in %s", TempDirBase, cwd),
				Detail:   err.Error(),
			})
			return
		}
		p.currentTmpDir = p.inputTmpDir
	} else {
		if err := os.MkdirAll(TempDirBase, 0700); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error creating temporary directory parent %s in %s", TempDirBase, cwd),
				Detail:   err.Error(),
			})
			return
		}
		p.currentTmpDir, err = ioutil.TempDir(TempDirBase, TempDirPattern)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error creating temporary directory %s in %s", TempDirBase, cwd),
				Detail:   err.Error(),
			})
			return
		}
	}

	for name, content := range p.files {
		perm, ok := p.perms[name]
		if !ok {
			perm = 0400
		}
		diags = append(diags, p.createFile(name, content, perm)...)
		if diags.HasError() {
			return
		}
	}
	return
}
func (p *program) prepareEnv() (env []string, diags diag.Diagnostics) {
	env = append(env, os.Environ()...)
	if len(p.currentTmpDir) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Cannot prepareEnv() because currentTmpDir is empty!",
		})
		return
	}

	env = append(env, fmt.Sprintf("%s=%s", "TF_EXTERNAL_DIR", p.currentTmpDir))

	if abs, err := filepath.Abs(p.currentTmpDir); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Failed converting %#v to absolute path!", p.currentTmpDir),
			Detail:   err.Error(),
		})
		return
	} else {
		env = append(env, fmt.Sprintf("%s=%s", "TF_EXTERNAL_DIR_ABS", abs))
	}

	var files []string
	for name, _ := range p.files {
		files = append(files, name)
	}
	env = append(env, fmt.Sprintf("%s=%s", "TF_EXTERNAL_MANAGED_FILES", strings.Join(files, ":")))
	return
}

func (p *program) executeCommand(key string) (diags diag.Diagnostics) {
	args := p.getArgs(key)
	cmd := exec.CommandContext(p.context, args[0], args[1:]...)
	env, d := p.prepareEnv()
	diags = append(diags, d...)
	if diags.HasError() {
		return
	}
	cmdRepr, _ := json.Marshal(args)

	cmd.Env = env
	output, err := cmd.CombinedOutput()
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("Combined output (%d bytes) of %s: %s", len(output), p.name, cmdRepr),
		Detail:   string(output),
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error when running %s", p.name),
			Detail:   fmt.Sprintf("ERROR=%v\nCOMMAND %v\nOUTPUT (%d bytes):\n%v", err.Error(), cmdRepr, len(output), string(output)),
		})
	}
	return
}

func (p *program) storeId() (diags diag.Diagnostics) {
	text, diags := p.readFile("id")
	if diags.HasError() {
		return
	}
	p.data.SetId(text)
	return
}

func (p *program) storeAttributes(attributes ...string) (diags diag.Diagnostics) {
	for _, attribute := range attributes {
		text, d := p.readFile(attribute)
		diags = append(d, diags...)
		if d.HasError() {
			continue
		}

		if err := p.data.Set(attribute, text); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error when setting %s attribute during %s", attribute, p.name),
				Detail:   err.Error(),
			})
		}
	}
	return
}

func (p *program) getArgs(key string) (spec []string) {
	programSpecV := p.data.Get(key).([]interface{})
	spec = make([]string, len(programSpecV))
	for i, obj := range programSpecV {
		spec[i] = obj.(string)
	}
	return spec
}

func (p *program) closeDir() (diags diag.Diagnostics) {
	if len(p.currentTmpDir) == 0 {
		return
	}
	if err := os.RemoveAll(p.currentTmpDir); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Error when cleaning up temporary directory %s", p.currentTmpDir),
			Detail:   err.Error(),
		})
	}
	p.currentTmpDir = ""
	return
}

func runProgram(ctx context.Context, data *schema.ResourceData, name string, commandKey string) (diags diag.Diagnostics) {
	p := Program(ctx, data)
	p.name = name
	diags = append(diags, p.openDir()...)
	if diags.HasError() {
		return
	}
	defer func() { diags = append(diags, p.closeDir()...) }()

	diags = append(diags, p.executeCommand(commandKey)...)
	if diags.HasError() {
		return
	}

	diags = append(diags, p.storeId()...)
	if diags.HasError() {
		return
	}

	diags = append(diags, p.storeAttributes("state", "output", "output_sensitive")...)
	if diags.HasError() {
		return
	}

	return
}

func (p *program) readFile(name string) (text string, diags diag.Diagnostics) {
	fullPath := path.Join(p.currentTmpDir, name)
	info, err := os.Stat(fullPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Error retrieving file information for %s", fullPath),
			Detail:   err.Error(),
		})
		return
	}
	var readMode os.FileMode = 0400
	oldMode := info.Mode()
	couldNotRead := (info.Mode() & 0400) == 0

	if couldNotRead {
		if err := os.Chmod(fullPath, readMode); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error when making file readable (%#o -> %#o) %s", oldMode, readMode, fullPath),
				Detail:   err.Error(),
			})
			return
		}
	}
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error opening file %s", fullPath),
			Detail:   err.Error(),
		})
	}
	text = string(content)
	if couldNotRead {
		if err := os.Chmod(fullPath, oldMode); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error when reverting file mode (%#o -> %#o) for %s", readMode, oldMode, fullPath),
				Detail:   err.Error(),
			})
			return
		}
	}
	return
}

func (p *program) createFile(name string, content string, perm os.FileMode) (diags diag.Diagnostics) {
	fullPath := path.Join(p.currentTmpDir, name)
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error creating file %s", fullPath),
			Detail:   err.Error(),
		})
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error when closing %s", fullPath),
				Detail:   err.Error(),
			})
		}
	}()

	if _, err := file.WriteString(content); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error writing to file %s", fullPath),
			Detail:   err.Error(),
		})
		return
	}
	return
}
