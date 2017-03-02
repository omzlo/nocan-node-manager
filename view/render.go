package view

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"github.com/yosssi/ace"
	"net/http"
	"pannetrat.com/nocan/clog"
)

/** CONTEXT **/

type Context struct {
	content  interface{}
	metadata map[string]interface{}
}

type ContextHolder interface {
	Content() interface{}
	MetaData() map[string]interface{}
	Flash() map[string][]string
	CommitSession(w http.ResponseWriter) error
}

func NewContext(r *http.Request, value interface{}) *Context {
	ctx := &Context{content: value, metadata: make(map[string]interface{})}
	ctx.LoadSession(r)
	ctx.metadata["request_uri"] = r.RequestURI
	ctx.metadata["self_url"] = r.URL.Path
	return ctx
}

func (ctx *Context) Content() interface{} {
	return ctx.content
}

func (ctx *Context) MetaData() map[string]interface{} {
	return ctx.metadata
}

func (ctx *Context) AddMetaData(key string, value interface{}) {
	ctx.metadata[key] = value
}

func (ctx *Context) GetMetaData(key string) interface{} {
	return ctx.metadata[key]
}

/** HTTPSESSION **/

type HttpSession map[string]interface{}

func (ctx *Context) CreateSession() HttpSession {
	hs := make(HttpSession)
	ctx.metadata["session"] = hs
	return hs
}

func (ctx *Context) getSession() HttpSession {
	hs, ok := ctx.metadata["session"]
	if !ok {
		return ctx.CreateSession()
	}
	return hs.(HttpSession)
}

func (ctx *Context) LoadSession(r *http.Request) error {
	hs := ctx.CreateSession()
	b64, err := r.Cookie("session")
	if err == nil {
		clog.Info("got cookie: %s", b64.Value)
		session, err := base64.URLEncoding.DecodeString(b64.Value)
		if err == nil {
			clog.Info("got glob: %v", session)
			dec := gob.NewDecoder(bytes.NewReader(session))
			err = dec.Decode(hs)
			clog.Info("got session: %+v", hs)
		}
	}
	return err
}

func (ctx *Context) DeleteSessionItem(key string) {
	hs := ctx.getSession()
	delete(hs, key)
}

func (ctx *Context) SetSessionItem(key string, value interface{}) {
	hs := ctx.getSession()
	hs[key] = value
}

func (ctx *Context) SessionItem(key string) (interface{}, bool) {
	hs := ctx.getSession()
	r, ok := hs[key]
	return r, ok
}

func (ctx *Context) AddFlashItem(key string, value string) {
	flash, ok := ctx.SessionItem("_flash")
	if !ok {
		flash = make(map[string][]string)
		ctx.SetSessionItem("_flash", flash)
	}

	f := flash.(map[string][]string)
	f[key] = append(f[key], value)
}

func (ctx *Context) Flash() map[string][]string {
	flash, ok := ctx.SessionItem("flash")
	if !ok {
		return make(map[string][]string)
	}
	return flash.(map[string][]string)
}

/*
func (ctx *Context) FlashItems(key string) []string {
	flash, ok := ctx.GetSessionItem("_flash")
	if !ok {
		return make([]string, 0)
	}
	return flash[key]
}
*/
func (ctx *Context) CommitSession(w http.ResponseWriter) error {
	var buf bytes.Buffer
	//clog.Info("%+v\n", ctx)
	hs := ctx.getSession()
	//clog.Info("%+v\n", hs)
	new_flash := hs["_flash"]
	old_flash := hs["flash"]
	delete(hs, "_flash")
	hs["flash"] = new_flash
	enc := gob.NewEncoder(&buf)
	hs["flash"] = old_flash
	if err := enc.Encode(hs); err != nil {
		return err
	}
	clog.Info("set glob: %v", buf.Bytes())
	b64 := base64.URLEncoding.EncodeToString(buf.Bytes())
	clog.Info("set b64: %s", b64)
	http.SetCookie(w, &http.Cookie{Name: "session", Value: b64})
	return nil
}

/** RENDER **/

func RenderJSON(w http.ResponseWriter, v ContextHolder) bool {
	js, err := json.MarshalIndent(v.Content(), "", "  ")
	if err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if v.CommitSession(w) != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(js); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func RenderAceTemplate(w http.ResponseWriter, base string, template string, context ContextHolder) bool {
	if err := context.CommitSession(w); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	tpl, err := ace.Load(base, template, &ace.Options{BaseDir: "../templates", Indent: "  ", DynamicReload: true})
	if err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if err := tpl.Execute(w, context); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func RedirectTo(w http.ResponseWriter, r *http.Request, where string, context ContextHolder) {
	if context != nil {
		if err := context.CommitSession(w); err != nil {
			LogHttpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, where, http.StatusSeeOther)
}

func RenderNotAcceptable(w http.ResponseWriter) {
	LogHttpError(w, "Sever does not have data in an acceptable format as defined by the 'Accept:' header.", http.StatusNotAcceptable)
}

func LogHttpError(w http.ResponseWriter, err string, code int) {
	http.Error(w, err, code)
	if code < 500 {
		clog.Warning("Returned HTTP status %d: %s", code, err)
	} else {
		clog.Error("Returned HTTP status %d: %s", code, err)
	}
}
