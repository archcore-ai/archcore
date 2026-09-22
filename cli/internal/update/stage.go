package update

import "errors"

// Stage names one step of the update pipeline. The five values are the whole
// vocabulary of the `stage` property on the `cli_update_failed` event
// (cli-update-telemetry.spec). A sixth value changes
// that contract and the queries built on it, so add one only with the spec.
type Stage string

const (
	StageCheck    Stage = "check"
	StageDownload Stage = "download"
	StageChecksum Stage = "checksum"
	StageExtract  Stage = "extract"
	StageReplace  Stage = "replace"
)

// StageError carries which stage of the update pipeline failed. It changes no
// message: Error delegates to the wrapped error, so user-visible output is
// byte-identical. Callers read the stage with errors.As.
type StageError struct {
	Stage Stage
	Err   error
}

func (e *StageError) Error() string {
	// stageErr never builds one without an error, but the fields are exported
	// and a stage-only value is a plausible mistake at a new failure point. A
	// named stage beats a nil dereference on an update error path.
	if e.Err == nil {
		return "update failed at stage " + string(e.Stage)
	}
	return e.Err.Error()
}

func (e *StageError) Unwrap() error { return e.Err }

// stageErr tags err with stage.
//
// It returns nil for a nil err, so a call site can wrap without a second
// branch. It also leaves an error that already carries a stage untouched: the
// innermost failure point is the accurate one, and a later re-tag would report
// the caller's stage for a failure that happened earlier.
func stageErr(stage Stage, err error) error {
	if err == nil {
		return nil
	}
	var already *StageError
	if errors.As(err, &already) {
		return err
	}
	return &StageError{Stage: stage, Err: err}
}

// StageOf reports the stage recorded in err, and whether one was found. It
// finds a stage through any wrapping chain that supports errors.As.
func StageOf(err error) (Stage, bool) {
	var se *StageError
	if errors.As(err, &se) {
		return se.Stage, true
	}
	return "", false
}
