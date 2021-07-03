package provider

import (
	"context"
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
	combinedOutput    bool
	context           context.Context
	currentTmpDir     string
	data              *schema.ResourceData
	files             map[string]string
	inputTmpDir       string
	keepTmpDir        bool
	keepTmpDirOnError bool
	name              string
	perms             map[string]os.FileMode
	providerData      *schema.ResourceData
	removeDir         bool
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

func Program(ctx context.Context, data *schema.ResourceData, providerData *schema.ResourceData) *program {
	oldStateV, newStateV := data.GetChange("state")
	p := &program{
		context:      ctx,
		data:         data,
		providerData: providerData,
	}
	p.inputTmpDir = data.Get("program_tmpdir").(string)
	p.keepTmpDir = data.Get("program_tmpdir_keep").(bool)
	p.keepTmpDirOnError = data.Get("program_tmpdir_keep_on_error").(bool)
	p.combinedOutput = data.Get("program_output_combined").(bool)

	p.files = map[string]string{
		"provider_input":   p.providerData.Get("input").(string),
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
		"stdout":           0600,
		"stderr":           0600,
		"stdall":           0600,
	}
	if p.combinedOutput {
		p.files["stdall"] = ""
		p.perms["stdall"] = 0600
	} else {
		p.files["stdout"] = ""
		p.perms["stdout"] = 0600

		p.files["stderr"] = ""
		p.perms["stderr"] = 0600
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
	for name := range p.files {
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
	var cmdReprLines []string
	for argNum, entry := range args {
		prefix := fmt.Sprintf("program[%d]: ", argNum)
		for argLineNum, line := range strings.Split(entry, "\n") {
			if argLineNum == 0 {
				line = prefix + line
			} else {
				line = strings.Repeat(" ", len(prefix)) + line
			}
			cmdReprLines = append(cmdReprLines, line)
		}
	}
	cmdRepr := strings.Join(cmdReprLines, "\n")

	cmd.Env = env
	var stdallPath string
	var stdoutPath string
	var stderrPath string
	if p.combinedOutput {
		var stdall *os.File
		var ds diag.Diagnostics
		stdall, stdallPath, ds = p.openFileForWriting("stdall")
		diags = append(diags, ds...)
		if diags.HasError() {
			return
		}
		defer func() { diags = append(diags, p.closeFile(stdall, diag.Warning)...) }()

		cmd.Stdout = stdall
		cmd.Stderr = stdall
	} else {
		var stdout *os.File
		var stderr *os.File
		var ds diag.Diagnostics
		stdout, stdoutPath, ds = p.openFileForWriting("stdout")
		diags = append(diags, ds...)
		if diags.HasError() {
			return
		}
		defer func() { diags = append(diags, p.closeFile(stdout, diag.Warning)...) }()

		stderr, stderrPath, ds = p.openFileForWriting("stderr")
		diags = append(diags, ds...)
		if diags.HasError() {
			return
		}
		defer func() { diags = append(diags, p.closeFile(stderr, diag.Warning)...) }()
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	err := cmd.Start()
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error when starting program %s in %s", p.name, p.currentTmpDir),
			Detail:   fmt.Sprintf("ERROR=%v\nCOMMAND:\n%s", err.Error(), cmdRepr),
		})
		return
	}

	err = cmd.Wait()

	var outputLength int
	var outputRepr string

	if p.combinedOutput {
		content, err := ioutil.ReadFile(stdallPath)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Error reading from combined output file %s for %s", stdallPath, p.name),
			})
		}

		outputLength = len(content)
		outputRepr = string(content)
	} else {
		stdoutOutput, err := ioutil.ReadFile(stdoutPath)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Error reading from stdout output file %s for %s", stdoutPath, p.name),
			})
		}

		stderrOutput, err := ioutil.ReadFile(stderrPath)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Error reading from stderr output file %s for %s", stderrPath, p.name),
			})
		}

		outputLength = len(stdoutOutput) + len(stderrOutput)
		outputRepr = fmt.Sprintf("=== STDOUT ===\n%s=== STDERR ===\n%s", string(stdoutOutput), string(stderrOutput))
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("Combined output (%d bytes) of %s:\n%s", outputLength, p.name, cmdRepr),
		Detail:   outputRepr, //
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error when running %s in %s: %v", p.name, p.currentTmpDir, err.Error()),
			Detail:   fmt.Sprintf("%s\nOUTPUT (%d bytes):\n%s", cmdRepr, outputLength, outputRepr),
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

func (p *program) closeDir(hadError bool) (diags diag.Diagnostics) {
	if p.currentTmpDir == "" {
		return
	}
	if p.keepTmpDir {
		return
	}
	if p.keepTmpDirOnError && hadError {
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
	err := ioutil.WriteFile(fullPath, []byte(content), perm)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error writing to file %s with permissions ", fullPath),
			Detail:   err.Error(),
		})
		return
	}
	return
}

func (p *program) openFileForWriting(name string) (file *os.File, fullPath string, diags diag.Diagnostics) {
	fullPath = filepath.Join(p.currentTmpDir, name)
	file, err := os.OpenFile(fullPath, os.O_APPEND, 0600)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error when opening file for writing %s", fullPath),
		})
	}
	return file, fullPath, diags
}

func (p *program) closeFile(file *os.File, severity diag.Severity) (diags diag.Diagnostics) {
	if file == nil {
		return
	}

	if err := file.Close(); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: severity,
			Summary:  fmt.Sprintf("Error when closing file for writing %s", file.Name()),
		})
	}
	return
}
