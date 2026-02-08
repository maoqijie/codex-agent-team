package codexrpc

// --- Constants for common values ---

// Approval policy values.
const (
	ApprovalPolicyUnlessTrusted = "untrusted"
	ApprovalPolicyOnFailure     = "on-failure"
	ApprovalPolicyOnRequest     = "on-request"
	ApprovalPolicyNever         = "never"
)

// Sandbox mode values.
const (
	SandboxReadOnly           = "read-only"
	SandboxWorkspaceWrite     = "workspace-write"
	SandboxDangerFullAccess   = "danger-full-access"
)

// Personality values.
const (
	PersonalityNone      = "none"
	PersonalityFriendly   = "friendly"
	PersonalityPragmatic  = "pragmatic"
)

// Approval decision values.
const (
	DecisionAccept           = "accept"
	DecisionAcceptForSession = "acceptForSession"
	DecisionDecline          = "decline"
	DecisionCancel           = "cancel"
)

// --- Initialize ---

// InitializeParams is sent as the first request to the app-server.
type InitializeParams struct {
	ClientInfo   ClientInfo              `json:"clientInfo"`
	Capabilities *InitializeCapabilities `json:"capabilities,omitempty"`
}

type ClientInfo struct {
	Name    string  `json:"name"`
	Title   *string `json:"title,omitempty"`
	Version string  `json:"version"`
}

type InitializeCapabilities struct {
	ExperimentalAPI bool `json:"experimentalApi"`
}

type InitializeResponse struct {
	UserAgent string `json:"userAgent"`
}

// --- Thread ---

// ThreadStartParams creates a new conversation thread.
type ThreadStartParams struct {
	Model                 *string `json:"model,omitempty"`
	ModelProvider         *string `json:"modelProvider,omitempty"`
	Cwd                   *string `json:"cwd,omitempty"`
	ApprovalPolicy        *string `json:"approvalPolicy,omitempty"`        // "untrusted"|"on-failure"|"on-request"|"never"
	Sandbox               *string `json:"sandbox,omitempty"`               // "read-only"|"workspace-write"|"danger-full-access"
	BaseInstructions      *string `json:"baseInstructions,omitempty"`
	DeveloperInstructions *string `json:"developerInstructions,omitempty"`
	Ephemeral             *bool   `json:"ephemeral,omitempty"`
	Personality           *string `json:"personality,omitempty"`           // "none"|"friendly"|"pragmatic"
}

type ThreadStartResponse struct {
	Thread         Thread `json:"thread"`
	Model          string `json:"model"`
	ModelProvider  string `json:"modelProvider"`
	Cwd            string `json:"cwd"`
	ApprovalPolicy string `json:"approvalPolicy"`
}

type Thread struct {
	ID            string  `json:"id"`
	Preview       string  `json:"preview"`
	ModelProvider string  `json:"modelProvider"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	Cwd           string  `json:"cwd"`
	CLIVersion    string  `json:"cliVersion"`
	Source        string  `json:"source"`
	Path          *string `json:"path,omitempty"`
}
