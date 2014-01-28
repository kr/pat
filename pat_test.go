package pat

import (
	"github.com/bmizerany/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
)

func TestPatMatch(t *testing.T) {
	params, ok := (&patHandler{"/", nil}).try("/")
	assert.Equal(t, true, ok)

	params, ok = (&patHandler{"/", nil}).try("/wrong_url")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/:name", nil}).try("/foo/bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar"}, "::match": {"/foo/bar"}}, params)

	params, ok = (&patHandler{"/foo/:name/baz", nil}).try("/foo/bar")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/:name/bar/", nil}).try("/foo/keith/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"keith"}, "::match": {"/foo/keith/bar/"}}, params)

	params, ok = (&patHandler{"/foo/:name/bar/", nil}).try("/foo/keith/bar/")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"keith"}, "::match": {"/foo/keith/bar/"}}, params)

	params, ok = (&patHandler{"/foo/:name/bar/", nil}).try("/foo/keith/bar")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/:name/baz", nil}).try("/foo/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar"}, "::match": {"/foo/bar/baz"}}, params)

	params, ok = (&patHandler{"/foo/:name/baz/:id", nil}).try("/foo/bar/baz")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/:name/baz/:id", nil}).try("/foo/bar/baz/123")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar"}, ":id": {"123"}, "::match": {"/foo/bar/baz/123"}}, params)

	params, ok = (&patHandler{"/foo/:name/baz/:name", nil}).try("/foo/bar/baz/123")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar", "123"}, "::match": {"/foo/bar/baz/123"}}, params)

	params, ok = (&patHandler{"/foo/:name.txt", nil}).try("/foo/bar.txt")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar"}, "::match": {"/foo/bar.txt"}}, params)

	params, ok = (&patHandler{"/foo/:name", nil}).try("/foo/:bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {":bar"}, "::match": {"/foo/:bar"}}, params)

	params, ok = (&patHandler{"/foo/:a:b", nil}).try("/foo/val1:val2")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":a": {"val1"}, ":b": {":val2"}, "::match": {"/foo/val1:val2"}}, params)

	params, ok = (&patHandler{"/foo/:a.", nil}).try("/foo/.")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":a": {""}, "::match": {"/foo/."}}, params)

	params, ok = (&patHandler{"/foo/:a:b", nil}).try("/foo/:bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":a": {""}, ":b": {":bar"}, "::match": {"/foo/:bar"}}, params)

	params, ok = (&patHandler{"/foo/:a:b:c", nil}).try("/foo/:bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":a": {""}, ":b": {""}, ":c": {":bar"}, "::match": {"/foo/:bar"}}, params)

	params, ok = (&patHandler{"/foo/::name", nil}).try("/foo/val1:val2")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":": {"val1"}, ":name": {":val2"}, "::match": {"/foo/val1:val2"}}, params)

	params, ok = (&patHandler{"/foo/:name.txt", nil}).try("/foo/bar/baz.txt")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/x:name", nil}).try("/foo/bar")
	assert.Equal(t, false, ok)

	params, ok = (&patHandler{"/foo/x:name", nil}).try("/foo/xbar")
	assert.Equal(t, true, ok)
	assert.Equal(t, url.Values{":name": {"bar"}, "::match": {"/foo/xbar"}}, params)
}

func TestPatRoutingHit(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.Query())
		assert.Equal(t, "keith", r.URL.Query().Get(":name"))
	}))

	r, err := http.NewRequest("GET", "/foo/keith?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTP(nil, r)

	assert.T(t, ok)
}

func TestPatRoutingMethodNotAllowed(t *testing.T) {
	p := New()

	var ok bool
	p.Post("/foo/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	p.Put("/foo/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	r, err := http.NewRequest("GET", "/foo/keith", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, r)

	assert.T(t, !ok)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

	allowed := strings.Split(rr.Header().Get("Allow"), ", ")
	sort.Strings(allowed)
	assert.Equal(t, allowed, []string{"POST", "PUT"})
}

// Check to make sure we don't pollute the Raw Query when we have no parameters
func TestPatNoParams(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.RawQuery)
		assert.Equal(t, "%3A%3Amatch=%2Ffoo%2F&", r.URL.RawQuery)
	}))

	r, err := http.NewRequest("GET", "/foo/", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTP(nil, r)

	assert.T(t, ok)
}

// Check to make sure we don't pollute the Raw Query when there are parameters but no pattern variables
func TestPatOnlyUserParams(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.RawQuery)
		assert.Equal(t, "%3A%3Amatch=%2Ffoo%2F&a=b", r.URL.RawQuery)
	}))

	r, err := http.NewRequest("GET", "/foo/?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTP(nil, r)

	assert.T(t, ok)
}

func TestPatImplicitRedirect(t *testing.T) {
	p := New()
	p.Get("/foo/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	r, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	p.ServeHTTP(res, r)

	if res.Code != 301 {
		t.Errorf("expected Code 301, was %d", res.Code)
	}

	if loc := res.Header().Get("Location"); loc != "/foo/" {
		t.Errorf("expected %q, got %q", "/foo/", loc)
	}

	p = New()
	p.Get("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	p.Get("/foo/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	r, err = http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	res = httptest.NewRecorder()
	res.Code = 200
	p.ServeHTTP(res, r)

	if res.Code != 200 {
		t.Errorf("expected Code 200, was %d", res.Code)
	}
}

func TestTail(t *testing.T) {
	if g := Tail("/:a/", "/x/y/z"); g != "y/z" {
		t.Fatalf("want %q, got %q", "y/z", g)
	}

	if g := Tail("/:a/", "/x"); g != "" {
		t.Fatalf("want %q, got %q", "", g)
	}

	if g := Tail("/:a/", "/x/"); g != "" {
		t.Fatalf("want %q, got %q", "", g)
	}

	if g := Tail("/:a", "/x/y/z"); g != "" {
		t.Fatalf("want: %q, got %q", "", g)
	}

	if g := Tail("/b/:a", "/x/y/z"); g != "" {
		t.Fatalf("want: %q, got %q", "", g)
	}
}

func TestStripPrefix(t *testing.T) {
	p := New()
	p.Get("/foo/:name/", StripPrefix(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/baz" {
			t.Errorf("Path = %s want /baz", r.URL.Path)
		}
	})))
	r, err := http.NewRequest("GET", "/foo/bar/baz", nil)
	if err != nil {
		t.Fatal(err)
	}
	p.ServeHTTP(nil, r)
}

func BenchmarkPatternMatching(b *testing.B) {
	p := New()
	p.Get("/hello/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		r, err := http.NewRequest("GET", "/hello/blake", nil)
		if err != nil {
			panic(err)
		}
		b.StartTimer()
		p.ServeHTTP(nil, r)
	}
}
