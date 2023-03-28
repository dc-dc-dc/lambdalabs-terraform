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
var _ resource.Resource = &SSHKeyResource{}
var _ resource.ResourceWithImportState = &SSHKeyResource{}

func NewSSHKeyResource() resource.Resource {
	return &SSHKeyResource{}
}

// SSHKeyResource defines the resource implementation.
type SSHKeyResource struct {
	apiKey string
}

// SSHKeyResourceModel describes the resource data model.
type SSHKeyResourceModel struct {
	Name       types.String `tfsdk:"name"`
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	Id         types.String `tfsdk:"id"`
}

type SSHKeyCreateRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key,omitempty"`
}

type SSHKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type SSHKeyCreateResponse struct {
	Data SSHKey `json:"data"`
}

type SSHKeyListResponse struct {
	Data []SSHKey `json:"data"`
}

func (r *SSHKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sshkey"
}

func (r *SSHKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the SSH key.",
				Required:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public key for the ssk key.",
				Optional:            true,
				Sensitive:           true,
			},
			"private_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Computed:            true,
				MarkdownDescription: "Private key for the SSH key. Only returned when generating a new key pair.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique Identifier (ID) of an SSH key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SSHKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSHKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *SSHKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	raw := SSHKeyCreateRequest{
		Name: data.Name.ValueString(),
	}
	if !data.PublicKey.IsNull() {
		raw.PublicKey = data.PublicKey.ValueString()
	}
	res, err := MakeAPICall(ctx, r.apiKey, http.MethodPost, "ssh-keys", raw)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
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
		return
	}

	var respData SSHKeyCreateResponse
	if err := json.NewDecoder(res.Body).Decode(&respData); err != nil {
		resp.Diagnostics.AddError("json error", err.Error())
		return
	}
	data.Id = types.StringValue(respData.Data.ID)
	data.Name = types.StringValue(respData.Data.Name)
	if respData.Data.PrivateKey != "" {
		data.PrivateKey = types.StringValue(respData.Data.PrivateKey)
	} else {
		data.PrivateKey = types.StringNull()
	}
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSHKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *SSHKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	res, err := MakeAPICall(ctx, r.apiKey, http.MethodGet, "ssh-keys", nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
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
		return
	}

	var respData SSHKeyListResponse
	if err := json.NewDecoder(res.Body).Decode(&respData); err != nil {
		resp.Diagnostics.AddError("json error", err.Error())
		return
	}
	// find the ssh-key in the list
	key := findKey(respData.Data, data.Id.ValueString())
	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.Id = types.StringValue(key.ID)
	data.Name = types.StringValue(key.Name)
	data.PublicKey = types.StringValue(key.PublicKey)
	if key.PrivateKey != "" {
		data.PrivateKey = types.StringValue(key.PrivateKey)
	} else {
		data.PrivateKey = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findKey(keys []SSHKey, id string) *SSHKey {
	for i := range keys {
		if keys[i].ID == id {
			return &keys[i]
		}
	}
	return nil
}

func (r *SSHKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *SSHKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSHKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *SSHKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	res, err := MakeAPICall(ctx, r.apiKey, http.MethodDelete, fmt.Sprintf("ssh-keys/%s", data.Id.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		var errData InstanceAPIErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errData); err != nil {
			resp.Diagnostics.AddError("json error", err.Error())
			return
		}
		resp.Diagnostics.AddError("client error", errData.Error.Message)
		return
	}
}

func (r *SSHKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
