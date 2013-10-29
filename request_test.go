package request

import (
    "testing"
    . "github.com/onsi/gomega"
    . "github.com/franela/goblin"
    "net/http/httptest"
    "net/http"
    "fmt"
    "strings"
    "time"
    "io"
)

func TestRequest(t *testing.T) {
    g := Goblin(t)

    RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

    g.Describe("Request", func() {

        g.Describe("General request methods", func() {
            var ts *httptest.Server

            g.Before(func() {
                ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    if (r.Method == "GET" || r.Method == "OPTIONS" || r.Method == "TRACE" || r.Method == "PATCH" || r.Method == "FOOBAR") && r.URL.Path == "/foo" {
                        w.WriteHeader(200)
                        fmt.Fprint(w, "bar")
                    }
                    if r.Method == "POST" && r.URL.Path == "/" {
                        w.Header().Add("Location", ts.URL + "/123")
                        w.WriteHeader(201)
                        io.Copy(w, r.Body)
                    }
                    if r.Method == "PUT" && r.URL.Path == "/foo/123" {
                        w.WriteHeader(200)
                        io.Copy(w, r.Body)
                    }
                    if r.Method == "DELETE" && r.URL.Path == "/foo/123" {
                        w.WriteHeader(204)
                    }
                }))
            })

            g.After(func() {
                ts.Close()
            })

            g.It("Should do a GET", func() {
                res, err := Request{ Uri: ts.URL + "/foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("bar"))
                Expect(res.StatusCode).Should(Equal(200))
            })

            g.Describe("POST", func() {
                g.It("Should send a string", func() {
                    res, err := Request{ Method: "POST", Uri: ts.URL, Body: "foo" }.Do()

                    Expect(err).Should(BeNil())
                    Expect(res.Body).Should(Equal("foo"))
                    Expect(res.StatusCode).Should(Equal(201))
                    Expect(res.Header.Get("Location")).Should(Equal(ts.URL + "/123"))
                })

                g.It("Should send a Reader", func() {
                    res, err := Request{ Method: "POST", Uri: ts.URL, Body: strings.NewReader("foo") }.Do()

                    Expect(err).Should(BeNil())
                    Expect(res.Body).Should(Equal("foo"))
                    Expect(res.StatusCode).Should(Equal(201))
                    Expect(res.Header.Get("Location")).Should(Equal(ts.URL + "/123"))
                })

                g.It("Send any object that is json encodable", func() {
                    obj := map[string]string {"foo": "bar"}
                    res, err := Request{ Method: "POST", Uri: ts.URL, Body: obj}.Do()

                    Expect(err).Should(BeNil())
                    Expect(res.Body).Should(Equal(`{"foo":"bar"}`))
                    Expect(res.StatusCode).Should(Equal(201))
                    Expect(res.Header.Get("Location")).Should(Equal(ts.URL + "/123"))
                })
            })

            g.It("Should do a PUT", func() {
                res, err := Request{ Method: "PUT", Uri: ts.URL + "/foo/123", Body: "foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("foo"))
                Expect(res.StatusCode).Should(Equal(200))
            })

            g.It("Should do a DELETE", func() {
                res, err := Request{ Method: "DELETE", Uri: ts.URL + "/foo/123" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.StatusCode).Should(Equal(204))
            })

            g.It("Should do a OPTIONS", func() {
                res, err := Request{ Method: "OPTIONS", Uri: ts.URL + "/foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("bar"))
                Expect(res.StatusCode).Should(Equal(200))
            })

            g.It("Should do a PATCH", func() {
                res, err := Request{ Method: "PATCH", Uri: ts.URL + "/foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("bar"))
                Expect(res.StatusCode).Should(Equal(200))
            })

            g.It("Should do a TRACE", func() {
                res, err := Request{ Method: "TRACE", Uri: ts.URL + "/foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("bar"))
                Expect(res.StatusCode).Should(Equal(200))
            })

            g.It("Should do a custom method", func() {
                res, err := Request{ Method: "FOOBAR", Uri: ts.URL + "/foo" }.Do()

                Expect(err).Should(BeNil())
                Expect(res.Body).Should(Equal("bar"))
                Expect(res.StatusCode).Should(Equal(200))
            })
        })

        g.Describe("Timeouts", func() {
            g.Describe("Connection timeouts", func() {
                g.It("Should connect timeout after a default of 1000 ms", func() {
                    start := time.Now()
                    res, err := Request{ Uri: "http://10.255.255.1" }.Do()
                    elapsed := time.Since(start)

                    Expect(elapsed).Should(BeNumerically("<", 1100 * time.Millisecond))
                    Expect(elapsed).Should(BeNumerically(">=", 1000 * time.Millisecond))
                    Expect(res).Should(BeNil())
                    Expect(err.ConnectTimeout()).Should(BeTrue())
                })
                g.It("Should connect timeout after a custom amount of time", func() {
                    SetConnectTimeout(100 * time.Millisecond)
                    start := time.Now()
                    res, err := Request{ Uri: "http://10.255.255.1" }.Do()
                    elapsed := time.Since(start)

                    Expect(elapsed).Should(BeNumerically("<", 150 * time.Millisecond))
                    Expect(elapsed).Should(BeNumerically(">=", 100 * time.Millisecond))
                    Expect(res).Should(BeNil())
                    Expect(err.ConnectTimeout()).Should(BeTrue())
                })
            })
            g.Describe("Request timeout", func() {
                var ts *httptest.Server
                stop := make(chan bool)

                g.Before(func() {
                    ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        <- stop
                        // just wait for someone to tell you when to end the request. this is used to simulate a slow server
                    }))
                })
                g.After(func() {
                    stop <- true
                    ts.Close()
                })
                g.It("Should request timeout after a custom amount of time", func() {
                    SetConnectTimeout(1000 * time.Millisecond)

                    start := time.Now()
                    res, err := Request{ Uri: ts.URL, Timeout: 500 * time.Millisecond }.Do()
                    elapsed := time.Since(start)

                    Expect(elapsed).Should(BeNumerically("<", 550 * time.Millisecond))
                    Expect(elapsed).Should(BeNumerically(">=", 500 * time.Millisecond))
                    Expect(res).Should(BeNil())
                    Expect(err.ConnectTimeout()).Should(BeFalse())
                    Expect(err.RequestTimeout()).Should(BeTrue())
                })
            })
        })

        g.Describe("Misc", func() {
            g.It("Should offer to set request headers")
        })
    })
}
