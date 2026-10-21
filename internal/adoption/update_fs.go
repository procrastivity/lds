package adoption

// UpdateFileState describes an eligible existing file at the update boundary.
type UpdateFileState struct {
	Content  string `json:"content"`
	SHA256   string `json:"sha256"`
	Bytes    int    `json:"bytes"`
	Identity string `json:"identity"`
	Mode     uint32 `json:"mode"`
	UID      uint32 `json:"uid"`
	GID      uint32 `json:"gid"`
	Links    uint64 `json:"links"`
	Metadata string `json:"metadata"`
}

// UpdateReplacement records effects even when replacement or verification fails.
type UpdateReplacement struct {
	Renamed      bool             `json:"renamed"`
	Observed     *UpdateFileState `json:"observed,omitempty"`
	Phase        string           `json:"phase"`
	Scratch      string           `json:"scratch,omitempty"`
	CleanupError string           `json:"cleanupError,omitempty"`
}
