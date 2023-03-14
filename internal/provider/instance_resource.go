package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &InstanceResource{}
var _ resource.ResourceWithImportState = &InstanceResource{}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	apiKey string
}

type InstanceResourceModel struct {
	RegionName       types.String `tfsdk:"region_name"`
	InstanceTypeName types.String `tfsdk:"instance_type_name"`
	SshKeyNames      types.List   `tfsdk:"ssh_key_names"`
	FileSystemNames  types.List   `tfsdk:"file_system_names"`
	// Quantity         types.Number `tfsdk:"quantity"`
	Name   types.String `tfsdk:"name"`
	IP     types.String `tfsdk:"ip"`
	Status types.String `tfsdk:"status"`
	Id     types.String `tfsdk:"id"`
}

type InstanceCreateAPIRequest struct {
	RegionName       string   `json:"region_name"`
	InstanceTypeName string   `json:"instance_type_name"`
	SSHKeyNames      []string `json:"ssh_key_names"`
	FileSystemNames  []string `json:"file_system_names,omitempty"`
	Quantity         int      `json:"Quantity"`
	Name             *string  `json:"name"`
}

type InstanceAPIErrorResponse struct {
	Error struct {
		Code       string  `json:"code"`
		Message    string  `json:"message"`
		Suggestion *string `json:"suggestion"`
	} `json:"error"`
}

type InstanceCreateAPIResponse struct {
	Data struct {
		InstanceIds []string `json:"instance_ids"`
	} `json:"data"`
}

type Instance struct {
	Id              string   `json:"id"`
	Name            string   `json:"name"`
	IP              string   `json:"ip"`
	Status          string   `json:"status"`
	SshKeyNames     []string `json:"ssh_key_names"`
	FileSystemNames []string `json:"file_system_names"`
	Region          struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"region"`
	InstanceType struct {
		Name             string      `json:"name"`
		Description      string      `json:"description"`
		PriceCentsHourly int         `json:"price_cents_per_hour"`
		Specs            interface{} `json:"specs"`
	} `json:"instance_type"`
	Hostname     string `json:"hostname"`
	JupyterToken string `json:"jupyter_token"`
	JupyterUrl   string `json:"jupyter_url"`
}

type InstanceGetAPIResponse struct {
	Data Instance `json:"data"`
}

type InstanceDeleteApiRequest struct {
	InstanceIds []string `json:"instance_ids"`
}

type InstanceDeleteApiResponse struct {
	Data struct {
		TerminatedInstances []Instance `json:"terminated_instances"`
	} `json:"data"`
}

func (r *InstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Instance resource",

		Attributes: map[string]schema.Attribute{
			"region_name": schema.StringAttribute{
				Required:    true,
				Description: "Short name of a region",
			},
			"instance_type_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of an instance type",
			},
			"ssh_key_names": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Names of the SSH keys to allow access to the instances. Currently, exactly one SSH key must be specified.",
			},
			"file_system_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of the file systems to attach to the instances. Currently, only one (if any) file system may be specified.",
			},
			// TODO: Add this back
			// "quantity": schema.NumberAttribute{
			// 	Optional:    true,
			// 	Description: "Number of instances to launch",
			// },
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "User-provided name for the instance",
			},
			"ip": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "ip address of the instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "description of the instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "id of the instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *InstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	apiKey, ok := req.ProviderData.(string)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.apiKey = apiKey
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *InstanceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	var sshKeys []string
	var fileSystemNames []string
	var name *string
	_ = data.SshKeyNames.ElementsAs(ctx, &sshKeys, false)
	if !data.FileSystemNames.IsNull() {
		_ = data.FileSystemNames.ElementsAs(ctx, &fileSystemNames, false)
	}
	if !data.Name.IsNull() {
		*name = data.Name.ValueString()
	}
	httpResp, err := MakeAPICall(ctx, r.apiKey, http.MethodPost, "instance-operations/launch", InstanceCreateAPIRequest{
		RegionName:       data.RegionName.ValueString(),
		InstanceTypeName: data.InstanceTypeName.ValueString(),
		SSHKeyNames:      sshKeys,
		Quantity:         1,
		FileSystemNames:  fileSystemNames,
		Name:             name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		var errData InstanceAPIErrorResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&errData); err != nil {
			resp.Diagnostics.AddError("json error", err.Error())
			return
		}
		resp.Diagnostics.AddError("client error", errData.Error.Message)
		return
	}

	var respData InstanceCreateAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&respData); err != nil {
		resp.Diagnostics.AddError("json error", err.Error())
		return
	}

	if len(respData.Data.InstanceIds) != 1 {
		resp.Diagnostics.AddError("resp error", fmt.Sprintf("expected 1 response got %d", len(respData.Data.InstanceIds)))
		return
	}
	data.IP = types.StringNull()
	data.Status = types.StringNull()
	data.Id = types.StringValue(respData.Data.InstanceIds[0])
	tflog.Trace(ctx, "created a resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *InstanceResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	res, err := MakeAPICall(ctx, r.apiKey, http.MethodGet, fmt.Sprintf("instances/%s", data.Id.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("resp error", err.Error())
		return
	}
	if res.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var errData InstanceAPIErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errData); err != nil {
			resp.Diagnostics.AddError("json error", err.Error())
			return
		}
		resp.Diagnostics.AddError("client error", errData.Error.Message)
		if res.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		return
	}

	var respData InstanceGetAPIResponse
	if err := json.NewDecoder(res.Body).Decode(&respData); err != nil {
		resp.Diagnostics.AddError("json error", err.Error())
		return
	}
	data.SshKeyNames, _ = types.ListValueFrom(ctx, types.StringType, respData.Data.SshKeyNames)
	data.InstanceTypeName = types.StringValue(respData.Data.InstanceType.Name)
	data.RegionName = types.StringValue(respData.Data.Region.Name)
	// data.IP = types.StringValue(respData.Data.IP)
	// data.Status = types.StringValue(respData.Data.Status)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *InstanceResource

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *InstanceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	res, err := MakeAPICall(ctx, r.apiKey, http.MethodPost, "instance-operations/terminate", InstanceDeleteApiRequest{
		InstanceIds: []string{data.Id.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError("resp error", err.Error())
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		var errData InstanceAPIErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errData); err != nil {
			resp.Diagnostics.AddError("json error", err.Error())
			return
		}
		resp.Diagnostics.AddError("client error", errData.Error.Message)
		return
	}

	var respData InstanceDeleteApiResponse
	if err := json.NewDecoder(res.Body).Decode(&respData); err != nil {
		resp.Diagnostics.AddError("json error", err.Error())
		return
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
