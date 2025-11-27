// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/internal/apiquery"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// FileService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewFileService] method instead.
type FileService struct {
	Options []option.RequestOption
}

// NewFileService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewFileService(opts ...option.RequestOption) (r FileService) {
	r = FileService{}
	r.Options = opts
	return
}

// List files and directories
func (r *FileService) List(ctx context.Context, query FileListParams, opts ...option.RequestOption) (res *[]FileListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "file"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get file status
func (r *FileService) GetStatus(ctx context.Context, query FileGetStatusParams, opts ...option.RequestOption) (res *[]FileGetStatusResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "file/status"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Read a file
func (r *FileService) Read(ctx context.Context, query FileReadParams, opts ...option.RequestOption) (res *FileReadResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "file/content"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type FileListResponse struct {
	Absolute string `json:"absolute,required"`
	Ignored  bool   `json:"ignored,required"`
	Name     string `json:"name,required"`
	Path     string `json:"path,required"`
	// Any of "file", "directory".
	Type FileListResponseType `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Absolute    respjson.Field
		Ignored     respjson.Field
		Name        respjson.Field
		Path        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileListResponse) RawJSON() string { return r.JSON.raw }
func (r *FileListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileListResponseType string

const (
	FileListResponseTypeFile      FileListResponseType = "file"
	FileListResponseTypeDirectory FileListResponseType = "directory"
)

type FileGetStatusResponse struct {
	Added   int64  `json:"added,required"`
	Path    string `json:"path,required"`
	Removed int64  `json:"removed,required"`
	// Any of "added", "deleted", "modified".
	Status FileGetStatusResponseStatus `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Added       respjson.Field
		Path        respjson.Field
		Removed     respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileGetStatusResponse) RawJSON() string { return r.JSON.raw }
func (r *FileGetStatusResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileGetStatusResponseStatus string

const (
	FileGetStatusResponseStatusAdded    FileGetStatusResponseStatus = "added"
	FileGetStatusResponseStatusDeleted  FileGetStatusResponseStatus = "deleted"
	FileGetStatusResponseStatusModified FileGetStatusResponseStatus = "modified"
)

type FileReadResponse struct {
	Content string        `json:"content,required"`
	Type    constant.Text `json:"type,required"`
	Diff    string        `json:"diff"`
	// Any of "base64".
	Encoding FileReadResponseEncoding `json:"encoding"`
	MimeType string                   `json:"mimeType"`
	Patch    FileReadResponsePatch    `json:"patch"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		Type        respjson.Field
		Diff        respjson.Field
		Encoding    respjson.Field
		MimeType    respjson.Field
		Patch       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileReadResponse) RawJSON() string { return r.JSON.raw }
func (r *FileReadResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileReadResponseEncoding string

const (
	FileReadResponseEncodingBase64 FileReadResponseEncoding = "base64"
)

type FileReadResponsePatch struct {
	Hunks       []FileReadResponsePatchHunk `json:"hunks,required"`
	NewFileName string                      `json:"newFileName,required"`
	OldFileName string                      `json:"oldFileName,required"`
	Index       string                      `json:"index"`
	NewHeader   string                      `json:"newHeader"`
	OldHeader   string                      `json:"oldHeader"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Hunks       respjson.Field
		NewFileName respjson.Field
		OldFileName respjson.Field
		Index       respjson.Field
		NewHeader   respjson.Field
		OldHeader   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileReadResponsePatch) RawJSON() string { return r.JSON.raw }
func (r *FileReadResponsePatch) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileReadResponsePatchHunk struct {
	Lines    []string `json:"lines,required"`
	NewLines float64  `json:"newLines,required"`
	NewStart float64  `json:"newStart,required"`
	OldLines float64  `json:"oldLines,required"`
	OldStart float64  `json:"oldStart,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Lines       respjson.Field
		NewLines    respjson.Field
		NewStart    respjson.Field
		OldLines    respjson.Field
		OldStart    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileReadResponsePatchHunk) RawJSON() string { return r.JSON.raw }
func (r *FileReadResponsePatchHunk) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileListParams struct {
	Path      string            `query:"path,required" json:"-"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [FileListParams]'s query parameters as `url.Values`.
func (r FileListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type FileGetStatusParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [FileGetStatusParams]'s query parameters as `url.Values`.
func (r FileGetStatusParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type FileReadParams struct {
	Path      string            `query:"path,required" json:"-"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [FileReadParams]'s query parameters as `url.Values`.
func (r FileReadParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
