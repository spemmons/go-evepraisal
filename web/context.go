package web

import (
	"html/template"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/evepraisal/go-evepraisal"
	"github.com/gorilla/sessions"
)

// Context contains all of the 'global' app context for the HTTP app
type Context struct {
	App            *evepraisal.App
	BaseURL        string
	ExtraJS        string
	AdBlock        string
	CookieStore    *sessions.CookieStore
	OauthConfig    *oauth2.Config
	OauthVerifyURL string

	templates map[string]*template.Template
	etags     map[string]string
}

// NewContext returns a new Context object given an app instance
func NewContext(app *evepraisal.App) *Context {
	ctx := &Context{App: app}
	ctx.GenerateStaticEtags()
	return ctx
}

func (ctx *Context) AccessToken(r *http.Request) string {
	return ctx.getSessionValueWithDefault(r, "access_token", "")
}