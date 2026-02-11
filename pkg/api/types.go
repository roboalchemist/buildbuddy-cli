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

func (j jsonInt64) Int64() int64 {
	return int64(j)
}

// --- Shared Types ---

// File represents an artifact file reference.
type File struct {
	Name      string    `json:"name,omitempty"`
	URI       string    `json:"uri,omitempty"`
	Hash      string    `json:"hash,omitempty"`
	SizeBytes jsonInt64 `json:"sizeBytes,omitempty"`
}

// InvocationMetadata represents a key-value metadata pair.
type InvocationMetadata struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Paginated is implemented by responses that support pagination.
type Paginated interface {
	GetNextPageToken() string
}

// --- GetInvocation ---

type GetInvocationRequest struct {
	Selector                *InvocationSelector `json:"selector,omitempty"`
	IncludeMetadata         bool                `json:"include_metadata,omitempty"`
	IncludeArtifacts        bool                `json:"include_artifacts,omitempty"`
	IncludeChildInvocations bool                `json:"include_child_invocations,omitempty"`
	PageToken               string              `json:"page_token,omitempty"`
}

func (r *GetInvocationRequest) SetPageToken(t string) { r.PageToken = t }

type InvocationSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
	CommitSHA    string `json:"commit_sha,omitempty"`
}

type GetInvocationResponse struct {
	Invocation    []Invocation `json:"invocation,omitempty"`
	NextPageToken string       `json:"nextPageToken,omitempty"`
}

func (r *GetInvocationResponse) GetNextPageToken() string { return r.NextPageToken }

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

type InvocationID struct {
	InvocationID string `json:"invocationId,omitempty"`
}

func (inv *Invocation) GetInvocationID() string {
	if inv.ID != nil {
		return inv.ID.InvocationID
	}
	return ""
}

type Tag struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// --- GetTarget ---

type GetTargetRequest struct {
	Selector  *TargetSelector `json:"selector,omitempty"`
	PageToken string          `json:"page_token,omitempty"`
}

func (r *GetTargetRequest) SetPageToken(t string) { r.PageToken = t }

type TargetSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
	TargetID     string `json:"target_id,omitempty"`
	Tag          string `json:"tag,omitempty"`
	Label        string `json:"label,omitempty"`
}

type GetTargetResponse struct {
	Target        []Target `json:"target,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

func (r *GetTargetResponse) GetNextPageToken() string { return r.NextPageToken }

type Target struct {
	ID       *TargetID `json:"id,omitempty"`
	Label    string    `json:"label,omitempty"`
	Status   string    `json:"status,omitempty"`
	Timing   *Timing   `json:"timing,omitempty"`
	RuleType string    `json:"ruleType,omitempty"`
	Tag      []string  `json:"tag,omitempty"`
	Language string    `json:"language,omitempty"`
}

type TargetID struct {
	InvocationID string `json:"invocationId,omitempty"`
	TargetID     string `json:"targetId,omitempty"`
}

type Timing struct {
	Duration string `json:"duration,omitempty"`
}

// --- GetLog ---

type GetLogRequest struct {
	Selector  *LogSelector `json:"selector,omitempty"`
	PageToken string       `json:"page_token,omitempty"`
}

func (r *GetLogRequest) SetPageToken(t string) { r.PageToken = t }

type LogSelector struct {
	InvocationID string `json:"invocation_id,omitempty"`
}

type GetLogResponse struct {
	Log           *Log   `json:"log,omitempty"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

func (r *GetLogResponse) GetNextPageToken() string { return r.NextPageToken }

type Log struct {
	Contents string `json:"contents,omitempty"`
}

// --- GetAction ---

type GetActionRequest struct {
	Selector  *ActionSelector `json:"selector,omitempty"`
	PageToken string          `json:"page_token,omitempty"`
}

func (r *GetActionRequest) SetPageToken(t string) { r.PageToken = t }

type ActionSelector struct {
	InvocationID    string `json:"invocation_id,omitempty"`
	TargetID        string `json:"target_id,omitempty"`
	ConfigurationID string `json:"configuration_id,omitempty"`
	ActionID        string `json:"action_id,omitempty"`
	TargetLabel     string `json:"target_label,omitempty"`
}

type GetActionResponse struct {
	Action        []Action `json:"action,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

func (r *GetActionResponse) GetNextPageToken() string { return r.NextPageToken }

type Action struct {
	ID          *ActionID `json:"id,omitempty"`
	File        []File    `json:"file,omitempty"`
	TargetLabel string    `json:"targetLabel,omitempty"`
	Shard       jsonInt64 `json:"shard,omitempty"`
	Run         jsonInt64 `json:"run,omitempty"`
	Attempt     jsonInt64 `json:"attempt,omitempty"`
}

type ActionID struct {
	InvocationID    string `json:"invocationId,omitempty"`
	TargetID        string `json:"targetId,omitempty"`
	ConfigurationID string `json:"configurationId,omitempty"`
	ActionID        string `json:"actionId,omitempty"`
}

// --- GetFile ---

type GetFileRequest struct {
	URI string `json:"uri,omitempty"`
}

type GetFileResponse struct {
	Data []byte `json:"data,omitempty"`
}

// --- DeleteFile ---

type DeleteFileRequest struct {
	URI string `json:"uri,omitempty"`
}

type DeleteFileResponse struct{}

// --- ExecuteWorkflow ---

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

type ExecuteWorkflowResponse struct {
	ActionStatuses []WorkflowActionStatus `json:"actionStatuses,omitempty"`
}

type WorkflowActionStatus struct {
	ActionName   string `json:"actionName,omitempty"`
	InvocationID string `json:"invocationId,omitempty"`
	Status       *RPCStatus `json:"status,omitempty"`
}

type RPCStatus struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// --- Run ---

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

type Step struct {
	Run string `json:"run,omitempty"`
}

type RunResponse struct {
	InvocationID string `json:"invocationId,omitempty"`
}
