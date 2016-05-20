package controller

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/remote"
	"github.com/CiscoCloud/drone/router/middleware/session"
	"github.com/CiscoCloud/drone/shared/crypto"
	"github.com/CiscoCloud/drone/shared/httputil"
	"github.com/CiscoCloud/drone/shared/token"
	"github.com/CiscoCloud/drone/store"
)

func PostRepo(c *gin.Context) {
	remote := remote.FromContext(c)
	user := session.User(c)
	owner := c.Param("owner")
	name := c.Param("name")
	paramNoActivate := c.Request.FormValue("no-activate")

	if user == nil {
		c.AbortWithStatus(403)
		return
	}

	r, err := remote.Repo(user, owner, name)
	if err != nil {
		c.String(404, err.Error())
		return
	}
	m, err := remote.Perm(user, owner, name)
	if err != nil {
		c.String(404, err.Error())
		return
	}
	if !m.Admin {
		c.String(403, "Administrative access is required.")
		return
	}

	// error if the repository already exists
	_, err = store.GetRepoOwnerName(c, owner, name)
	if err == nil {
		c.String(409, "Repository already exists.")
		return
	}

	// set the repository owner to the
	// currently authenticated user.
	r.UserID = user.ID
	r.AllowPush = true
	r.AllowPull = true
	r.Timeout = 60 // 1 hour default build time
	r.Hash = crypto.Rand()

	// crates the jwt token used to verify the repository
	t := token.New(token.HookToken, r.FullName)
	sig, err := t.Sign(r.Hash)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	link := fmt.Sprintf(
		"%s/hook?access_token=%s",
		httputil.GetURL(c.Request),
		sig,
	)

	// generate an RSA key and add to the repo
	key, err := crypto.GeneratePrivateKey()
	if err != nil {
		c.String(500, err.Error())
		return
	}
	keys := new(model.Key)
	keys.Public = string(crypto.MarshalPublicKey(&key.PublicKey))
	keys.Private = string(crypto.MarshalPrivateKey(key))

	var noActivate bool
	noActivate, err = strconv.ParseBool(paramNoActivate)
	if err != nil {
		noActivate = false
	}
	if !noActivate {
		// activate the repository before we make any
		// local changes to the database.
		err = remote.Activate(user, r, keys, link)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
	}

	// persist the repository
	err = store.CreateRepo(c, r)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	keys.RepoID = r.ID
	err = store.CreateKey(c, keys)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.JSON(200, r)
}

func PatchRepo(c *gin.Context) {
	repo := session.Repo(c)
	user := session.User(c)

	in := &struct {
		IsTrusted   *bool  `json:"trusted,omitempty"`
		Timeout     *int64 `json:"timeout,omitempty"`
		AllowPull   *bool  `json:"allow_pr,omitempty"`
		AllowPush   *bool  `json:"allow_push,omitempty"`
		AllowDeploy *bool  `json:"allow_deploy,omitempty"`
		AllowTag    *bool  `json:"allow_tag,omitempty"`
	}{}
	if err := c.Bind(in); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if in.AllowPush != nil {
		repo.AllowPush = *in.AllowPush
	}
	if in.AllowPull != nil {
		repo.AllowPull = *in.AllowPull
	}
	if in.AllowDeploy != nil {
		repo.AllowDeploy = *in.AllowDeploy
	}
	if in.AllowTag != nil {
		repo.AllowTag = *in.AllowTag
	}
	if in.IsTrusted != nil && user.Admin {
		repo.IsTrusted = *in.IsTrusted
	}
	if in.Timeout != nil && user.Admin {
		repo.Timeout = *in.Timeout
	}

	err := store.UpdateRepo(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, repo)
}

func GetRepo(c *gin.Context) {
	repo := session.Repo(c)
	user := session.User(c)
	if user == nil {
		c.IndentedJSON(http.StatusOK, repo)
		return
	}

	repoResp := struct {
		*model.Repo
		Token string `json:"hook_token,omitempty"`
	}{repo, ""}
	if user.Admin {
		t := token.New(token.HookToken, repo.FullName)
		sig, err := t.Sign(repo.Hash)
		if err != nil {
			log.Errorf("Error creating hook token: %s", err)
		} else {
			repoResp.Token = sig
		}
	}

	c.IndentedJSON(http.StatusOK, repoResp)
}

func GetRepoKey(c *gin.Context) {
	repo := session.Repo(c)
	keys, err := store.GetKey(c, repo)
	if err != nil {
		c.String(404, "Error fetching repository key")
	} else {
		c.String(http.StatusOK, keys.Public)
	}
}

func PostRepoKey(c *gin.Context) {
	repo := session.Repo(c)
	keys, err := store.GetKey(c, repo)
	if err != nil {
		c.String(404, "Error fetching repository key")
		return
	}
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.String(500, "Error reading private key from body. %s", err)
		return
	}
	pkey := crypto.UnmarshalPrivateKey(body)
	if pkey == nil {
		c.String(500, "Cannot unmarshal private key. Invalid format.")
		return
	}

	keys.Public = string(crypto.MarshalPublicKey(&pkey.PublicKey))
	keys.Private = string(crypto.MarshalPrivateKey(pkey))

	err = store.UpdateKey(c, keys)
	if err != nil {
		c.String(500, "Error updating repository key")
		return
	}
	c.String(201, keys.Public)
}

func DeleteRepo(c *gin.Context) {
	remote := remote.FromContext(c)
	repo := session.Repo(c)
	user := session.User(c)

	err := store.DeleteRepo(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	remote.Deactivate(user, repo, httputil.GetURL(c.Request))
	c.Writer.WriteHeader(http.StatusOK)
}

func PostSecure(c *gin.Context) {
	repo := session.Repo(c)

	in, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// we found some strange characters included in
	// the yaml file when entered into a browser textarea.
	// these need to be removed
	in = bytes.Replace(in, []byte{'\xA0'}, []byte{' '}, -1)

	// make sure the Yaml is valid format to prevent
	// a malformed value from being used in the build
	err = yaml.Unmarshal(in, &yaml.MapSlice{})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	key, err := store.GetKey(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// encrypts using go-jose
	out, err := crypto.Encrypt(string(in), key.Private)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, out)
}

func PostReactivate(c *gin.Context) {

}
