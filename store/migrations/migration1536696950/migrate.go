package migration1536696950

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/smartcontractkit/chainlink/store/migrations/migration0"
	"github.com/smartcontractkit/chainlink/store/models"
	"github.com/smartcontractkit/chainlink/store/orm"
	null "gopkg.in/guregu/null.v3"
)

type Migration struct{}

func (m Migration) Timestamp() string {
	return "1536696950"
}

func (m Migration) Migrate(orm *orm.ORM) error {
	var jrs []migration0.JobRun
	if err := orm.All(&jrs); err != nil {
		var latestJrs []JobRun
		if err := orm.All(&latestJrs); err != nil {
			return fmt.Errorf("failed migration1536696950: %v", err)
		}
		return nil
	}

	tx, err := orm.Begin(true)
	if err != nil {
		return fmt.Errorf("error starting transaction: %+v", err)
	}
	defer tx.Rollback()

	for _, jr := range jrs {
		jr2 := m.Convert(jr)
		if err := tx.Save(&jr2); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m Migration) Convert(jr migration0.JobRun) JobRun {
	return JobRun{
		ID:             jr.ID,
		JobID:          jr.JobID,
		Result:         convertRunResult(jr.Result),
		Status:         jr.Status,
		TaskRuns:       convertTaskRuns(jr.TaskRuns),
		CreatedAt:      jr.CreatedAt,
		Initiator:      jr.Initiator,
		CreationHeight: jr.CreationHeight,
		Overrides:      convertRunResult(jr.Overrides),
	}
}

func convertRunResult(rr migration0.RunResult) RunResult {
	return RunResult{
		JobRunID:     rr.JobRunID,
		Data:         rr.Data,
		Status:       rr.Status,
		ErrorMessage: rr.ErrorMessage,
		Amount:       (*migration0.Link)(rr.Amount),
	}
}

func convertTaskRuns(oldTRs []migration0.TaskRun) []TaskRun {
	var trs []TaskRun
	for _, otr := range oldTRs {
		trs = append(trs, TaskRun{
			ID:     otr.ID,
			Result: convertRunResult(otr.Result),
			Status: otr.Status,
			Task:   otr.Task,
		})
	}
	return trs
}

type JobRun struct {
	ID             string               `json:"id" storm:"id,unique"`
	JobID          string               `json:"jobId" storm:"index"`
	Result         RunResult            `json:"result" storm:"inline"`
	Status         migration0.RunStatus `json:"status" storm:"index"`
	TaskRuns       []TaskRun            `json:"taskRuns" storm:"inline"`
	CreatedAt      time.Time            `json:"createdAt" storm:"index"`
	CompletedAt    null.Time            `json:"completedAt"`
	Initiator      migration0.Initiator `json:"initiator"`
	CreationHeight *hexutil.Big         `json:"creationHeight"`
	Overrides      RunResult            `json:"overrides"`
}

type TaskRun struct {
	ID     string               `json:"id" storm:"id,unique"`
	Result RunResult            `json:"result"`
	Status migration0.RunStatus `json:"status"`
	Task   migration0.TaskSpec  `json:"task"`
}

type RunResult struct {
	JobRunID     string               `json:"jobRunId"`
	Data         models.JSON          `json:"data"`
	Status       migration0.RunStatus `json:"status"`
	ErrorMessage null.String          `json:"error"`
	Amount       *migration0.Link     `json:"amount,omitempty"`
}
