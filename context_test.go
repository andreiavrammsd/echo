package echo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	Template struct {
		templates *template.Template
	}
)

func (t *Template) Render(w io.Writer, name string, data interface{}, c Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func TestContext(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(userJSON))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec).(*context)

	assert := assert.New(t)

	// Echo
	assert.Equal(e, c.Echo())

	// Request
	assert.NotNil(c.Request())

	// Response
	assert.NotNil(c.Response())

	//--------
	// Render
	//--------

	tmpl := &Template{
		templates: template.Must(template.New("hello").Parse("Hello, {{.}}!")),
	}
	c.echo.Renderer = tmpl
	err := c.Render(http.StatusOK, "hello", "Jon Snow")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("Hello, Jon Snow!", rec.Body.String())
	}

	c.echo.Renderer = nil
	err = c.Render(http.StatusOK, "hello", "Jon Snow")
	assert.Error(err)

	// JSON
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSON, rec.Body.String())
	}

	// JSON with "?pretty"
	req = httptest.NewRequest(http.MethodGet, "/?pretty", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSONPretty, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil) // reset

	// JSONPretty
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSONPretty(http.StatusOK, user{1, "Jon Snow"}, "  ")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSONPretty, rec.Body.String())
	}

	// JSON (error)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, make(chan bool))
	assert.Error(err)

	// JSONP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	callback := "callback"
	err = c.JSONP(http.StatusOK, callback, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJavaScriptCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(callback+"("+userJSON+");", rec.Body.String())
	}

	// XML
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.XML(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(xml.Header+userXML, rec.Body.String())
	}

	// XML with "?pretty"
	req = httptest.NewRequest(http.MethodGet, "/?pretty", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.XML(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(xml.Header+userXMLPretty, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)

	// XML (error)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.XML(http.StatusOK, make(chan bool))
	assert.Error(err)

	// XMLPretty
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.XMLPretty(http.StatusOK, user{1, "Jon Snow"}, "  ")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(xml.Header+userXMLPretty, rec.Body.String())
	}

	// String
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.String(http.StatusOK, "Hello, World!")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMETextPlainCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal("Hello, World!", rec.Body.String())
	}

	// HTML
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.HTML(http.StatusOK, "Hello, <strong>World!</strong>")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMETextHTMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal("Hello, <strong>World!</strong>", rec.Body.String())
	}

	// Stream
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	r := strings.NewReader("response from a stream")
	err = c.Stream(http.StatusOK, "application/octet-stream", r)
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("application/octet-stream", rec.Header().Get(HeaderContentType))
		assert.Equal("response from a stream", rec.Body.String())
	}

	// Attachment
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.Attachment("_fixture/images/walle.png", "walle.png")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("attachment; filename=\"walle.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(219885, rec.Body.Len())
	}

	// Inline
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.Inline("_fixture/images/walle.png", "walle.png")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("inline; filename=\"walle.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(219885, rec.Body.Len())
	}

	// NoContent
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	c.NoContent(http.StatusOK)
	assert.Equal(http.StatusOK, rec.Code)

	// Error
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	c.Error(errors.New("error"))
	assert.Equal(http.StatusInternalServerError, rec.Code)

	// Reset
	c.SetParamNames("foo")
	c.SetParamValues("bar")
	c.Set("foe", "ban")
	c.query = url.Values(map[string][]string{"fon": {"baz"}})
	c.Reset(req, httptest.NewRecorder())
	assert.Equal(0, len(c.ParamValues()))
	assert.Equal(0, len(c.ParamNames()))
	assert.Equal(0, len(c.store))
	assert.Equal("", c.Path())
	assert.Equal(0, len(c.QueryParams()))
}

func TestContextCookie(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	theme := "theme=light"
	user := "user=Jon Snow"
	req.Header.Add(HeaderCookie, theme)
	req.Header.Add(HeaderCookie, user)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec).(*context)

	assert := assert.New(t)

	// Read single
	cookie, err := c.Cookie("theme")
	if assert.NoError(err) {
		assert.Equal("theme", cookie.Name)
		assert.Equal("light", cookie.Value)
	}

	// Read multiple
	for _, cookie := range c.Cookies() {
		switch cookie.Name {
		case "theme":
			assert.Equal("light", cookie.Value)
		case "user":
			assert.Equal("Jon Snow", cookie.Value)
		}
	}

	// Write
	cookie = &http.Cookie{
		Name:     "SSID",
		Value:    "Ap4PGTEq",
		Domain:   "labstack.com",
		Path:     "/",
		Expires:  time.Now(),
		Secure:   true,
		HttpOnly: true,
	}
	c.SetCookie(cookie)
	assert.Contains(rec.Header().Get(HeaderSetCookie), "SSID")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "Ap4PGTEq")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "labstack.com")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "Secure")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "HttpOnly")
}

func TestContextPath(t *testing.T) {
	e := New()
	r := e.Router()

	r.Add(http.MethodGet, "/users/:id", nil)
	c := e.NewContext(nil, nil)
	r.Find(http.MethodGet, "/users/1", c)

	assert := assert.New(t)

	assert.Equal("/users/:id", c.Path())

	r.Add(http.MethodGet, "/users/:uid/files/:fid", nil)
	c = e.NewContext(nil, nil)
	r.Find(http.MethodGet, "/users/1/files/1", c)
	assert.Equal("/users/:uid/files/:fid", c.Path())
}

func TestContextPathParam(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, nil)

	// ParamNames
	c.SetParamNames("uid", "fid")
	assert.EqualValues(t, []string{"uid", "fid"}, c.ParamNames())

	// ParamValues
	c.SetParamValues("101", "501")
	assert.EqualValues(t, []string{"101", "501"}, c.ParamValues())

	// Param
	assert.Equal(t, "501", c.Param("fid"))
}

func TestContextPathIntParam(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, nil)

	c.SetParamNames("id", "dir", "notint")
	c.SetParamValues("1", "-1", "1a")

	// IntParam
	assert.Equal(t, 1, c.IntParam("id"))
	assert.Equal(t, -1, c.IntParam("dir"))
	assert.Equal(t, 0, c.IntParam("notint"))
	assert.Equal(t, 0, c.IntParam("notset"))
}

func TestContextFormValue(t *testing.T) {
	f := make(url.Values)
	f.Set("name", "Jon Snow")
	f.Set("email", "jon@labstack.com")

	e := New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(f.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationForm)
	c := e.NewContext(req, nil)

	// FormValue
	assert.Equal(t, "Jon Snow", c.FormValue("name"))
	assert.Equal(t, "jon@labstack.com", c.FormValue("email"))

	// FormParams
	params, err := c.FormParams()
	if assert.NoError(t, err) {
		assert.Equal(t, url.Values{
			"name":  []string{"Jon Snow"},
			"email": []string{"jon@labstack.com"},
		}, params)
	}
}

func TestContextQueryParam(t *testing.T) {
	q := make(url.Values)
	q.Set("name", "Jon Snow")
	q.Set("email", "jon@labstack.com")
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	e := New()
	c := e.NewContext(req, nil)

	// QueryParam
	assert.Equal(t, "Jon Snow", c.QueryParam("name"))
	assert.Equal(t, "jon@labstack.com", c.QueryParam("email"))

	// QueryParams
	assert.Equal(t, url.Values{
		"name":  []string{"Jon Snow"},
		"email": []string{"jon@labstack.com"},
	}, c.QueryParams())
}

func TestContextFormFile(t *testing.T) {
	e := New()
	buf := new(bytes.Buffer)
	mr := multipart.NewWriter(buf)
	w, err := mr.CreateFormFile("file", "test")
	if assert.NoError(t, err) {
		w.Write([]byte("test"))
	}
	mr.Close()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Set(HeaderContentType, mr.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.FormFile("file")
	if assert.NoError(t, err) {
		assert.Equal(t, "test", f.Filename)
	}
}

func TestContextMultipartForm(t *testing.T) {
	e := New()
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	mw.WriteField("name", "Jon Snow")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Set(HeaderContentType, mw.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.MultipartForm()
	if assert.NoError(t, err) {
		assert.NotNil(t, f)
	}
}

func TestContextRedirect(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	assert.Equal(t, nil, c.Redirect(http.StatusMovedPermanently, "http://labstack.github.io/echo"))
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "http://labstack.github.io/echo", rec.Header().Get(HeaderLocation))
	assert.Error(t, c.Redirect(310, "http://labstack.github.io/echo"))
}

func TestContextStore(t *testing.T) {
	var c Context
	c = new(context)
	c.Set("name", "Jon Snow")
	assert.Equal(t, "Jon Snow", c.Get("name"))
}

func TestContextHandler(t *testing.T) {
	e := New()
	r := e.Router()
	b := new(bytes.Buffer)

	r.Add(http.MethodGet, "/handler", func(Context) error {
		_, err := b.Write([]byte("handler"))
		return err
	})
	c := e.NewContext(nil, nil)
	r.Find(http.MethodGet, "/handler", c)
	c.Handler()(c)
	assert.Equal(t, "handler", b.String())
}
