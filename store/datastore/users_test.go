package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func Test_userstore(t *testing.T) {
	db := openTest()
	defer db.Close()
	s := From(db)

	g := goblin.Goblin(t)
	g.Describe("User", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM users")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM jobs")
		})

		g.It("Should Update a User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err1 := s.Users().Create(&user)
			err2 := s.Users().Update(&user)
			getuser, err3 := s.Users().Get(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
		})

		g.It("Should Add a new User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err := s.Users().Create(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := model.User{
				Login:  "joe",
				Token:  "f0b461ca586c27872b43a0685cbc2847",
				Secret: "976f22a5eef7caacb7e678d6c52f49b1",
				Email:  "foo@bar.com",
				Avatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Active: true,
				Admin:  true,
			}

			s.Users().Create(&user)
			getuser, err := s.Users().Get(user.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Token).Equal(getuser.Token)
			g.Assert(user.Secret).Equal(getuser.Secret)
			g.Assert(user.Email).Equal(getuser.Email)
			g.Assert(user.Avatar).Equal(getuser.Avatar)
			g.Assert(user.Active).Equal(getuser.Active)
			g.Assert(user.Admin).Equal(getuser.Admin)
		})

		g.It("Should Get a User By Login", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.Users().Create(&user)
			getuser, err := s.Users().GetLogin(user.Login)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
		})

		g.It("Should Enforce Unique User Login", func() {
			user1 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := s.Users().Create(&user1)
			err2 := s.Users().Create(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := model.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.Users().Create(&user1)
			s.Users().Create(&user2)
			users, err := s.Users().GetList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Get a User Count", func() {
			user1 := model.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.Users().Create(&user1)
			s.Users().Create(&user2)
			count, err := s.Users().Count()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(2)
		})

		g.It("Should Get a User Count Zero", func() {
			count, err := s.Users().Count()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(0)
		})

		g.It("Should Del a User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.Users().Create(&user)
			_, err1 := s.Users().Get(user.ID)
			err2 := s.Users().Delete(&user)
			_, err3 := s.Users().Get(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should get the Build feed for a User", func() {
			repo1 := &model.Repo{
				UserID:   1,
				Owner:    "bradrydzewski",
				Name:     "drone",
				FullName: "bradrydzewski/drone",
			}
			repo2 := &model.Repo{
				UserID:   2,
				Owner:    "drone",
				Name:     "drone",
				FullName: "drone/drone",
			}
			repo3 := &model.Repo{
				UserID:   2,
				Owner:    "octocat",
				Name:     "hello-world",
				FullName: "octocat/hello-world",
			}
			s.Repos().Create(repo1)
			s.Repos().Create(repo2)
			s.Repos().Create(repo3)

			build1 := &model.Build{
				RepoID: repo1.ID,
				Status: model.StatusFailure,
			}
			build2 := &model.Build{
				RepoID: repo1.ID,
				Status: model.StatusSuccess,
			}
			build3 := &model.Build{
				RepoID: repo2.ID,
				Status: model.StatusSuccess,
			}
			build4 := &model.Build{
				RepoID: repo3.ID,
				Status: model.StatusSuccess,
			}
			s.Builds().Create(build1)
			s.Builds().Create(build2)
			s.Builds().Create(build3)
			s.Builds().Create(build4)

			builds, err := s.Users().GetFeed([]*model.RepoLite{
				{FullName: "bradrydzewski/drone"},
				{FullName: "drone/drone"},
			})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(3)
			g.Assert(builds[0].FullName).Equal(repo2.FullName)
			g.Assert(builds[1].FullName).Equal(repo1.FullName)
			g.Assert(builds[2].FullName).Equal(repo1.FullName)
		})
	})
}
