package domain

import "time"

type JobStatus string

const (
	JobAccepted     JobStatus = "accepted"
	JobRunning      JobStatus = "running"
	JobAwaitingAuth JobStatus = "awaiting_auth"
	JobProcessing   JobStatus = "processing"
	JobPublished    JobStatus = "published"
	JobFailed       JobStatus = "failed"
	JobCancelled    JobStatus = "cancelled"
)

type Visibility string

const (
	VisibilityPrivate   Visibility = "private"
	VisibilityUnlisted  Visibility = "unlisted"
	VisibilityPublic    Visibility = "public"
)

type MediaRef struct {
	LocalPath string    `json:"local_path,omitempty"`
	S3        *S3Object `json:"s3,omitempty"`
}

type S3Object struct {
	Bucket          string `json:"bucket,omitempty"`
	Key             string `json:"key,omitempty"`
	PresignedGetURL string `json:"presigned_get_url,omitempty"`
}

type Audience struct {
	MadeForKids bool `json:"made_for_kids"`
}

type CreateJobRequest struct {
	ProfileID       string     `json:"profile_id"`
	Video           MediaRef   `json:"video"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Tags            []string   `json:"tags,omitempty"`
	Thumbnail       *MediaRef  `json:"thumbnail,omitempty"`
	Visibility      Visibility `json:"visibility"`
	ScheduleAt      *time.Time `json:"schedule_at,omitempty"`
	Category        string     `json:"category,omitempty"`
	GameTitle       string     `json:"game_title,omitempty"`
	Audience        Audience   `json:"audience"`
	ClientRequestID string     `json:"client_request_id,omitempty"`
}

type Job struct {
	ID              string     `json:"job_id"`
	IdempotencyKey  string     `json:"idempotency_key,omitempty"`
	Status          JobStatus  `json:"status"`
	CurrentStep     string     `json:"current_step,omitempty"`
	ProfileID       string     `json:"profile_id"`
	Request         CreateJobRequest `json:"request"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	ArtifactDir     string     `json:"artifact_dir,omitempty"`
	ScreenshotPath  string     `json:"screenshot_path,omitempty"`
	TracePath       string     `json:"trace_path,omitempty"`
	YouTubeVideoURL string     `json:"youtube_video_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}
