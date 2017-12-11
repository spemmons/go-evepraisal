package web

import (
	"html/template"
	"net/http"
	"time"

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

func (ctx *Context) OauthClient(r *http.Request) *http.Client {
	token := new(oauth2.Token)
	token.AccessToken = ctx.getSessionValueWithDefault(r, "access_token", "")
	token.RefreshToken = ctx.getSessionValueWithDefault(r, "refresh_token", "")
	token.TokenType = ctx.getSessionValueWithDefault(r, "token_type", "")
	token.Expiry, _ = time.Parse(time.RFC3339,ctx.getSessionValue(r, "expiry").(string))

	newToken, err := ctx.OauthConfig.TokenSource(r.Context(), token).Token()
	if err == nil {
		token = newToken
	}

	client := ctx.OauthConfig.Client(r.Context(),token)

	return client
}