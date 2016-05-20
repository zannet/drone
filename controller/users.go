package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/router/middleware/session"
	"github.com/CiscoCloud/drone/shared/crypto"
	"github.com/CiscoCloud/drone/shared/token"
	"github.com/CiscoCloud/drone/store"
)

func GetUsers(c *gin.Context) {
	users, err := store.GetUserList(c)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		tokenstr = ""
	}
	userWithToken := struct {
		*model.User
		Token string `json:"token,omitempty"`
	}{user, tokenstr}

	c.IndentedJSON(http.StatusOK, userWithToken)
}

func PatchUser(c *gin.Context) {
	me := session.User(c)
	in := &struct {
		model.User
		Token string `json:"token,omitempty"`
	}{}
	err := c.Bind(in)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	user.Admin = in.Admin
	user.Token = in.Token
	user.Active = in.Active

	// cannot update self
	if me.ID == user.ID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err = store.UpdateUser(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func PostUser(c *gin.Context) {
	in := &struct {
		model.User
		Token string `json:"token,omitempty"`
	}{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	user := &model.User{}
	user.Login = in.Login
	user.Email = in.Email
	user.Admin = in.Admin
	user.Token = in.Token
	user.Avatar = in.Avatar
	user.Active = true
	user.Hash = crypto.Rand()

	err = store.CreateUser(c, user)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	me := session.User(c)

	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// cannot delete self
	if me.ID == user.ID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err = store.DeleteUser(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}
