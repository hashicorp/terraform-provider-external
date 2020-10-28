package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceExternal() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceExternalCreate,
		ReadContext:   resourceExternalRead,
		UpdateContext: resourceExternalUpdate,
		DeleteContext: resourceExternalDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"program_tmpdir": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"program_tmpdir_keep": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"program_tmpdir_keep_on_error": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"program_output_combined": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"program_create": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				MinItems: 1,
			},
			"program_read": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
				MinItems: 1,
			},
			"program_update": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
				MinItems: 1,
			},
			"program_delete": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
				MinItems: 1,
			},
			"input": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"input_sensitive": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				Default:   "",
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"output": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"output_sensitive": {
				Type:      schema.TypeString,
				Sensitive: true,
				Computed:  true,
			},
		},
	}
}

func resourceExternalCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	p := Program(ctx, data)
	p.name = "create"
	diags = append(diags, p.openDir()...)
	defer func() { diags = append(diags, p.closeDir(diags.HasError())...) }()
	if diags.HasError() {
		return
	}
	if _, ok := data.GetOk("program_create"); ok {
		p.name += ">create"
		diags = append(diags, p.executeCommand("program_create")...)
		if diags.HasError() {
			return
		}
	} else {
		p.name += ">update"
		diags = append(diags, p.executeCommand("program_update")...)
	}
	p.name = "create"

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

func resourceExternalRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	p := Program(ctx, data)
	p.name = "read"
	diags = append(diags, p.openDir()...)
	defer func() { diags = append(diags, p.closeDir(diags.HasError())...) }()
	if diags.HasError() {
		return
	}

	diags = append(diags, p.executeCommand("program_read")...)
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

func resourceExternalUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	p := Program(ctx, data)
	name := "update"
	p.name = name
	diags = append(diags, p.openDir()...)
	defer func() { diags = append(diags, p.closeDir(diags.HasError())...) }()
	if diags.HasError() {
		return
	}
	diags = append(diags, p.executeCommand("program_update")...)
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

func resourceExternalDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	diags = append(diags, runProgram(ctx, data, "delete", "program_delete")...)

	data.SetId("")
	return
}
