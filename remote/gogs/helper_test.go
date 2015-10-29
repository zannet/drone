package gogs

import (
	"bytes"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/gogs/testdata"

	"github.com/franela/goblin"
	"github.com/gogits/go-gogs-client"
)

func Test_parse(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Gogs", func() {

		g.It("Should parse push hook payload", func() {
			buf := bytes.NewBufferString(testdata.PushHook)
			hook, err := parsePush(buf)
			g.Assert(err == nil).IsTrue()
			g.Assert(hook.Ref).Equal("refs/heads/master")
			g.Assert(hook.After).Equal("ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Before).Equal("4b2626259b5a97b6b4eab5e6cca66adb986b672b")
			g.Assert(hook.Compare).Equal("http://gogs.golang.org/gordon/hello-world/compare/4b2626259b5a97b6b4eab5e6cca66adb986b672b...ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.Url).Equal("http://gogs.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.Owner.Name).Equal("gordon")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Owner.Username).Equal("gordon")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Pusher.Name).Equal("gordon")
			g.Assert(hook.Pusher.Email).Equal("gordon@golang.org")
			g.Assert(hook.Pusher.Username).Equal("gordon")
			g.Assert(hook.Sender.Login).Equal("gordon")
			g.Assert(hook.Sender.Avatar).Equal("http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
		})

		g.It("Should return a Build struct from a push hook", func() {
			buf := bytes.NewBufferString(testdata.PushHook)
			hook, _ := parsePush(buf)
			build := buildFromPush(hook)
			g.Assert(build.Event).Equal(model.EventPush)
			g.Assert(build.Commit).Equal(hook.After)
			g.Assert(build.Ref).Equal(hook.Ref)
			g.Assert(build.Link).Equal(hook.Compare)
			g.Assert(build.Branch).Equal("master")
			g.Assert(build.Message).Equal(hook.Commits[0].Message)
			g.Assert(build.Avatar).Equal("//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
			g.Assert(build.Author).Equal(hook.Sender.Login)

		})

		g.It("Should return a Repo struct from a push hook", func() {
			buf := bytes.NewBufferString(testdata.PushHook)
			hook, _ := parsePush(buf)
			repo := repoFromPush(hook)
			g.Assert(repo.Name).Equal(hook.Repo.Name)
			g.Assert(repo.Owner).Equal(hook.Repo.Owner.Username)
			g.Assert(repo.FullName).Equal("gordon/hello-world")
			g.Assert(repo.Link).Equal(hook.Repo.Url)
		})

		g.It("Should return a Perm struct from a Gogs Perm", func() {
			perms := []gogs.Permission{
				{true, true, true},
				{true, true, false},
				{true, false, false},
			}
			for _, from := range perms {
				perm := toPerm(from)
				g.Assert(perm.Pull).Equal(from.Pull)
				g.Assert(perm.Push).Equal(from.Push)
				g.Assert(perm.Admin).Equal(from.Admin)
			}
		})

		g.It("Should return a Repo struct from a Gogs Repo", func() {
			from := gogs.Repository{
				FullName: "gophers/hello-world",
				Owner: gogs.User{
					UserName:  "gordon",
					AvatarUrl: "//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				CloneUrl: "http://gogs.golang.org/gophers/hello-world.git",
				HtmlUrl:  "http://gogs.golang.org/gophers/hello-world",
				Private:  true,
			}
			repo := toRepo(&from)
			g.Assert(repo.FullName).Equal(from.FullName)
			g.Assert(repo.Owner).Equal(from.Owner.UserName)
			g.Assert(repo.Name).Equal("hello-world")
			g.Assert(repo.Branch).Equal("master")
			g.Assert(repo.Link).Equal(from.HtmlUrl)
			g.Assert(repo.Clone).Equal(from.CloneUrl)
			g.Assert(repo.Avatar).Equal(from.Owner.AvatarUrl)
			g.Assert(repo.IsPrivate).Equal(from.Private)
		})

		g.It("Should return a RepoLite struct from a Gogs Repo", func() {
			from := gogs.Repository{
				FullName: "gophers/hello-world",
				Owner: gogs.User{
					UserName:  "gordon",
					AvatarUrl: "//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
			}
			repo := toRepoLite(&from)
			g.Assert(repo.FullName).Equal(from.FullName)
			g.Assert(repo.Owner).Equal(from.Owner.UserName)
			g.Assert(repo.Name).Equal("hello-world")
			g.Assert(repo.Avatar).Equal(from.Owner.AvatarUrl)
		})

		g.It("Should correct a malformed avatar url", func() {

			var urls = []struct {
				Before string
				After  string
			}{
				{
					"http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"http://gogs.golang.org/avatars/1",
					"http://gogs.golang.org/avatars/1",
				},
				{
					"http://gogs.golang.org//avatars/1",
					"http://gogs.golang.org/avatars/1",
				},
			}

			for _, url := range urls {
				got := fixMalformedAvatar(url.Before)
				g.Assert(got).Equal(url.After)
			}
		})

		g.It("Should expand the avatar url", func() {
			var urls = []struct {
				Before string
				After  string
			}{
				{
					"/avatars/1",
					"http://gogs.io/avatars/1",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
			}

			var repo = "http://gogs.io/foo/bar"
			for _, url := range urls {
				got := expandAvatar(repo, url.Before)
				g.Assert(got).Equal(url.After)
			}
		})
	})
}
