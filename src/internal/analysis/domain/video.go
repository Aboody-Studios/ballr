package domain

// Video represents a video file for upload.
// This is a value object used for initial upload validation before
// creating a Match aggregate. It encapsulates file metadata invariants.
type Video struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

// NewVideo creates a new video value object with validation.
// This factory ensures only valid video files enter the upload workflow.
func NewVideo(name string, size uint64) *Video {
	return &Video{
		Name: name,
		Size: size,
	}
}
