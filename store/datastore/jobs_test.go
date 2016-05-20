package datastore

import (
	"testing"

	"github.com/CiscoCloud/drone/model"
	"github.com/franela/goblin"
)

func Test_jobstore(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Job", func() {

		// before each test we purge the package table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM jobs")
			db.Exec("DELETE FROM builds")
		})

		g.It("Should Set a job", func() {
			job := &model.Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 0,
				Number:   1,
			}
			err1 := s.Jobs().Create(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			job.Status = "started"
			err2 := s.Jobs().Update(job)
			g.Assert(err2 == nil).IsTrue()

			getjob, err3 := s.Jobs().Get(job.ID)
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a Job by ID", func() {
			job := &model.Job{
				BuildID:     1,
				Status:      "pending",
				ExitCode:    1,
				Number:      1,
				Environment: map[string]string{"foo": "bar"},
			}
			err1 := s.Jobs().Create(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := s.Jobs().Get(job.ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
			g.Assert(getjob.ExitCode).Equal(job.ExitCode)
			g.Assert(getjob.Environment).Equal(job.Environment)
			g.Assert(getjob.Environment["foo"]).Equal("bar")
		})

		g.It("Should Get a Job by Number", func() {
			job := &model.Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 1,
				Number:   1,
			}
			err1 := s.Jobs().Create(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := s.Jobs().GetNumber(&model.Build{ID: 1}, 1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a List of Jobs by Commit", func() {

			build := model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
			}
			jobs := []*model.Job{
				&model.Job{
					BuildID:  1,
					Status:   "success",
					ExitCode: 0,
					Number:   1,
				},
				&model.Job{
					BuildID:  3,
					Status:   "error",
					ExitCode: 1,
					Number:   2,
				},
				&model.Job{
					BuildID:  5,
					Status:   "pending",
					ExitCode: 0,
					Number:   3,
				},
			}

			err1 := s.Builds().Create(&build, jobs...)
			g.Assert(err1 == nil).IsTrue()
			getjobs, err2 := s.Jobs().GetList(&build)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(len(getjobs)).Equal(3)
			g.Assert(getjobs[0].Number).Equal(1)
			g.Assert(getjobs[0].Status).Equal(model.StatusSuccess)
		})
	})
}
