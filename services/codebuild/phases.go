package codebuild

import "time"

// BuildPhaseType represents a CodeBuild build phase.
type BuildPhaseType string

const (
	PhaseSubmitted        BuildPhaseType = "SUBMITTED"
	PhaseQueued           BuildPhaseType = "QUEUED"
	PhaseProvisioning     BuildPhaseType = "PROVISIONING"
	PhaseDownloadSource   BuildPhaseType = "DOWNLOAD_SOURCE"
	PhaseInstall          BuildPhaseType = "INSTALL"
	PhasePreBuild         BuildPhaseType = "PRE_BUILD"
	PhaseBuild            BuildPhaseType = "BUILD"
	PhasePostBuild        BuildPhaseType = "POST_BUILD"
	PhaseUploadArtifacts  BuildPhaseType = "UPLOAD_ARTIFACTS"
	PhaseFinalizing       BuildPhaseType = "FINALIZING"
	PhaseCompleted        BuildPhaseType = "COMPLETED"
)

// BuildPhase represents the state of a single build phase.
type BuildPhase struct {
	PhaseType     BuildPhaseType
	PhaseStatus   string // SUCCEEDED, FAILED, TIMED_OUT, IN_PROGRESS
	StartTime     time.Time
	EndTime       *time.Time
	DurationInSec int64
}

// AllPhases returns the ordered list of build phases that a build progresses through.
func AllPhases() []BuildPhaseType {
	return []BuildPhaseType{
		PhaseSubmitted,
		PhaseQueued,
		PhaseProvisioning,
		PhaseDownloadSource,
		PhaseInstall,
		PhasePreBuild,
		PhaseBuild,
		PhasePostBuild,
		PhaseUploadArtifacts,
		PhaseFinalizing,
		PhaseCompleted,
	}
}

// AllPhaseNames returns phase names as strings (for log generation).
func AllPhaseNames() []string {
	phases := AllPhases()
	names := make([]string, len(phases))
	for i, p := range phases {
		names[i] = string(p)
	}
	return names
}

// GenerateCompletedPhases creates a set of build phases as if the build
// completed successfully. Each phase gets a start/end time offset from baseTime.
func GenerateCompletedPhases(baseTime time.Time) []BuildPhase {
	phases := AllPhases()
	result := make([]BuildPhase, len(phases))
	offset := time.Duration(0)

	for i, pt := range phases {
		start := baseTime.Add(offset)
		var duration time.Duration
		switch pt {
		case PhaseSubmitted:
			duration = 0
		case PhaseQueued:
			duration = 100 * time.Millisecond
		case PhaseProvisioning:
			duration = 500 * time.Millisecond
		case PhaseDownloadSource:
			duration = 200 * time.Millisecond
		case PhaseInstall:
			duration = 300 * time.Millisecond
		case PhasePreBuild:
			duration = 100 * time.Millisecond
		case PhaseBuild:
			duration = 1 * time.Second
		case PhasePostBuild:
			duration = 100 * time.Millisecond
		case PhaseUploadArtifacts:
			duration = 200 * time.Millisecond
		case PhaseFinalizing:
			duration = 100 * time.Millisecond
		case PhaseCompleted:
			duration = 0
		}

		end := start.Add(duration)
		result[i] = BuildPhase{
			PhaseType:     pt,
			PhaseStatus:   "SUCCEEDED",
			StartTime:     start,
			EndTime:       &end,
			DurationInSec: int64(duration.Seconds()),
		}
		offset += duration
	}

	return result
}

// GenerateFailedPhases creates phases where the BUILD phase fails.
func GenerateFailedPhases(baseTime time.Time) []BuildPhase {
	phases := GenerateCompletedPhases(baseTime)
	for i := range phases {
		if phases[i].PhaseType == PhaseBuild {
			phases[i].PhaseStatus = "FAILED"
			// Truncate remaining phases
			for j := i + 1; j < len(phases); j++ {
				if phases[j].PhaseType != PhaseCompleted {
					phases[j].PhaseStatus = "SKIPPED"
				} else {
					phases[j].PhaseStatus = "SUCCEEDED"
				}
			}
			break
		}
	}
	return phases
}
