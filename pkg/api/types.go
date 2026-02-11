// Package api provides request/response types for the BuildBuddy Enterprise API.
// All types use proto3 JSON conventions: camelCase field names, int64 as strings.
package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// jsonInt64 handles proto3 JSON int64 values which come as strings.
type jsonInt64 int64

func (j *jsonInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*j = jsonInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*j = jsonInt64(n)
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into int64", string(data))
}

func (j jsonInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(j))
}

// Int64 returns the underlying int64 value.
func (j jsonInt64) Int64() int64 {
	return int64(j)
}

// --- Shared Types ---

// File represents an artifact file reference returned by the API.
type File struct {
	Name      string    `json:"name,omitempty"`
	URI       string    `json:"uri,omitempty"`
	Hash      string    `json:"hash,omitempty"`
	SizeBytes jsonInt64 `json:"sizeBytes,omitempty"`
}

// InvocationMetadata represents a key-value metadata pair on an invocation.
type InvocationMetadata struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Paginated is implemented by responses that support pagination.
type Paginated interface {
	GetNextPageToken() string
}

// --- GetInvocation ---

// GetInvocationRequest is the request body for the GetInvocation API endpoint.
type GetInvocationRequest struct {
	Selector                *InvocationSelector `json:"selector,omitempty"`
	IncludeMetadata         bool                `json:"include_metadata,omitempty"`
	IncludeArtifacts        bool                `json:"include_artifacts,omitempty"`
	IncludeChildInvocations bool                `json:"include_child_invocations,omitempty"`
	PageToken               string              `json:"page_token,omitempty"`
}

// SetPageToken sets the pagination token for the next page of results.
func (r *GetInvocationRequest) SetPageToken(t string) { r.PageToken = t }

// InvocationSelector selects which invocations to retrieve (by ID or commit SHA).
type InvocationSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
	CommitSHA    string `json:"commit_sha,omitempty"`
}

// GetInvocationResponse is the response body from the GetInvocation API endpoint.
type GetInvocationResponse struct {
	Invocation    []Invocation `json:"invocation,omitempty"`
	NextPageToken string       `json:"nextPageToken,omitempty"`
}

// GetNextPageToken returns the token for the next page of results.
func (r *GetInvocationResponse) GetNextPageToken() string { return r.NextPageToken }

// Invocation represents a single BuildBuddy build invocation.
type Invocation struct {
	ID               *InvocationID        `json:"id,omitempty"`
	Success          bool                 `json:"success,omitempty"`
	User             string               `json:"user,omitempty"`
	DurationUsec     jsonInt64            `json:"durationUsec,omitempty"`
	Host             string               `json:"host,omitempty"`
	Command          string               `json:"command,omitempty"`
	Pattern          string               `json:"pattern,omitempty"`
	ActionCount      jsonInt64            `json:"actionCount,omitempty"`
	CreatedAtUsec    jsonInt64            `json:"createdAtUsec,omitempty"`
	UpdatedAtUsec    jsonInt64            `json:"updatedAtUsec,omitempty"`
	RepoURL          string               `json:"repoUrl,omitempty"`
	BranchName       string               `json:"branchName,omitempty"`
	CommitSHA        string               `json:"commitSha,omitempty"`
	Role             string               `json:"role,omitempty"`
	InvocationStatus string               `json:"invocationStatus,omitempty"`
	BazelExitCode    string               `json:"bazelExitCode,omitempty"`
	BuildMetadata    []InvocationMetadata `json:"buildMetadata,omitempty"`
	WorkspaceStatus  []InvocationMetadata `json:"workspaceStatus,omitempty"`
	Artifacts        []File               `json:"artifacts,omitempty"`
	ChildInvocations []InvocationID       `json:"childInvocations,omitempty"`
	Tags             []Tag                `json:"tags,omitempty"`
	ConsoleBuffer    string               `json:"consoleBuffer,omitempty"`
}

// InvocationID wraps the unique identifier for a build invocation.
type InvocationID struct {
	InvocationID string `json:"invocationId,omitempty"`
}

// GetInvocationID returns the string ID, or empty if the ID is nil.
func (inv *Invocation) GetInvocationID() string {
	if inv.ID != nil {
		return inv.ID.InvocationID
	}
	return ""
}

// Tag represents a key-value tag on an invocation.
type Tag struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// --- GetTarget ---

// GetTargetRequest is the request body for the GetTarget API endpoint.
type GetTargetRequest struct {
	Selector  *TargetSelector `json:"selector,omitempty"`
	PageToken string          `json:"page_token,omitempty"`
}

// SetPageToken sets the pagination token for the next page of results.
func (r *GetTargetRequest) SetPageToken(t string) { r.PageToken = t }

// TargetSelector selects which targets to retrieve.
type TargetSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
	TargetID     string `json:"target_id,omitempty"`
	Tag          string `json:"tag,omitempty"`
	Label        string `json:"label,omitempty"`
}

// GetTargetResponse is the response body from the GetTarget API endpoint.
type GetTargetResponse struct {
	Target        []Target `json:"target,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// GetNextPageToken returns the token for the next page of results.
func (r *GetTargetResponse) GetNextPageToken() string { return r.NextPageToken }

// Target represents a build target within an invocation.
type Target struct {
	ID       *TargetID `json:"id,omitempty"`
	Label    string    `json:"label,omitempty"`
	Status   string    `json:"status,omitempty"`
	Timing   *Timing   `json:"timing,omitempty"`
	RuleType string    `json:"ruleType,omitempty"`
	Tag      []string  `json:"tag,omitempty"`
	Language string    `json:"language,omitempty"`
}

// TargetID wraps the unique identifier for a build target.
type TargetID struct {
	InvocationID string `json:"invocationId,omitempty"`
	TargetID     string `json:"targetId,omitempty"`
}

// Timing represents execution timing information for a target.
type Timing struct {
	Duration string `json:"duration,omitempty"`
}

// --- GetLog ---

// GetLogRequest is the request body for the GetLog API endpoint.
type GetLogRequest struct {
	Selector  *LogSelector `json:"selector,omitempty"`
	PageToken string       `json:"page_token,omitempty"`
}

// SetPageToken sets the pagination token for the next page of results.
func (r *GetLogRequest) SetPageToken(t string) { r.PageToken = t }

// LogSelector selects the log to retrieve by invocation ID.
type LogSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
}

// GetLogResponse is the response body from the GetLog API endpoint.
type GetLogResponse struct {
	Log           *Log   `json:"log,omitempty"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// GetNextPageToken returns the token for the next page of results.
func (r *GetLogResponse) GetNextPageToken() string { return r.NextPageToken }

// Log represents the contents of a build log.
type Log struct {
	Contents string `json:"contents,omitempty"`
}

// --- GetAction ---

// GetActionRequest is the request body for the GetAction API endpoint.
type GetActionRequest struct {
	Selector  *ActionSelector `json:"selector,omitempty"`
	PageToken string          `json:"page_token,omitempty"`
}

// SetPageToken sets the pagination token for the next page of results.
func (r *GetActionRequest) SetPageToken(t string) { r.PageToken = t }

// ActionSelector selects which actions to retrieve.
type ActionSelector struct {
	InvocationID    string `json:"invocation_id,omitempty"`
	TargetID        string `json:"target_id,omitempty"`
	ConfigurationID string `json:"configuration_id,omitempty"`
	ActionID        string `json:"action_id,omitempty"`
	TargetLabel     string `json:"target_label,omitempty"`
}

// GetActionResponse is the response body from the GetAction API endpoint.
type GetActionResponse struct {
	Action        []Action `json:"action,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// GetNextPageToken returns the token for the next page of results.
func (r *GetActionResponse) GetNextPageToken() string { return r.NextPageToken }

// Action represents a build action within a target.
type Action struct {
	ID          *ActionID `json:"id,omitempty"`
	File        []File    `json:"file,omitempty"`
	TargetLabel string    `json:"targetLabel,omitempty"`
	Shard       jsonInt64 `json:"shard,omitempty"`
	Run         jsonInt64 `json:"run,omitempty"`
	Attempt     jsonInt64 `json:"attempt,omitempty"`
}

// ActionID wraps the unique identifier for a build action.
type ActionID struct {
	InvocationID    string `json:"invocationId,omitempty"`
	TargetID        string `json:"targetId,omitempty"`
	ConfigurationID string `json:"configurationId,omitempty"`
	ActionID        string `json:"actionId,omitempty"`
}

// --- GetFile ---

// GetFileRequest is the request body for the GetFile API endpoint.
type GetFileRequest struct {
	URI string `json:"uri,omitempty"`
}

// GetFileResponse is the response body containing file data from GetFile.
type GetFileResponse struct {
	Data []byte `json:"data,omitempty"`
}

// --- DeleteFile ---

// DeleteFileRequest is the request body for the DeleteFile API endpoint.
type DeleteFileRequest struct {
	URI string `json:"uri,omitempty"`
}

// DeleteFileResponse is the response body from DeleteFile (empty on success).
type DeleteFileResponse struct{}

// --- ExecuteWorkflow ---

// ExecuteWorkflowRequest is the request body for the ExecuteWorkflow API endpoint.
type ExecuteWorkflowRequest struct {
	RepoURL      string            `json:"repo_url,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	CommitSHA    string            `json:"commit_sha,omitempty"`
	ActionNames  []string          `json:"action_names,omitempty"`
	Visibility   string            `json:"visibility,omitempty"`
	Async        bool              `json:"async,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	DisableRetry bool              `json:"disable_retry,omitempty"`
}

// ExecuteWorkflowResponse is the response body from ExecuteWorkflow.
type ExecuteWorkflowResponse struct {
	ActionStatuses []WorkflowActionStatus `json:"actionStatuses,omitempty"`
}

// WorkflowActionStatus represents the result of a triggered workflow action.
type WorkflowActionStatus struct {
	ActionName   string     `json:"actionName,omitempty"`
	InvocationID string     `json:"invocationId,omitempty"`
	Status       *RPCStatus `json:"status,omitempty"`
}

// RPCStatus represents a gRPC-style status code and message.
type RPCStatus struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// --- Run ---

// RunRequest is the request body for the Run API endpoint (remote execution).
type RunRequest struct {
	Repo                    string            `json:"repo,omitempty"`
	Branch                  string            `json:"branch,omitempty"`
	CommitSHA               string            `json:"commit_sha,omitempty"`
	Steps                   []Step            `json:"steps,omitempty"`
	Env                     map[string]string `json:"env,omitempty"`
	PlatformProperties      map[string]string `json:"platform_properties,omitempty"`
	RemoteHeaders           []string          `json:"remote_headers,omitempty"`
	Timeout                 string            `json:"timeout,omitempty"`
	UseSystemGitCredentials bool              `json:"use_system_git_credentials,omitempty"`
	SkipAutoCheckout        bool              `json:"skip_auto_checkout,omitempty"`
}

// Step represents a single build step in a Run request.
type Step struct {
	Run string `json:"run,omitempty"`
}

// RunResponse is the response body from the Run API endpoint.
type RunResponse struct {
	InvocationID string `json:"invocationId,omitempty"`
}
