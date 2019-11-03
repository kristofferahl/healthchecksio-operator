package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	logr "github.com/go-logr/logr"
	healthchecksio "github.com/kristofferahl/go-healthchecksio"
)

// FakeServerResponse holds a response string and status code
type FakeServerResponse struct {
	StatusCode   int
	ResponseBody string
}

// NewTestHealthchecksioServer creates a new fake HTTP server
func NewTestHealthchecksioServer(responses ...*FakeServerResponse) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var fr *FakeServerResponse
		if len(responses) > 0 {
			for i := len(responses)/2 - 1; i >= 0; i-- {
				opp := len(responses) - 1 - i
				responses[i], responses[opp] = responses[opp], responses[i]
			}

			fr, responses = responses[len(responses)-1], responses[:len(responses)-1]
		}

		if fr != nil {
			res.WriteHeader(fr.StatusCode)
			res.Write([]byte(fr.ResponseBody))
			return
		}

		http.Error(res, "No more FakeServerResponse registered", http.StatusInternalServerError)
	}))

	return server
}

// NewTestHealthchecksioClient creates a new Healthchecksio Client, configured for testing purposes
func NewTestHealthchecksioClient(t *testing.T, apiKey, baseURL string) *healthchecksio.Client {
	hc := healthchecksio.NewClient(apiKey)
	if baseURL != "" {
		hc.BaseURL = baseURL
	}
	hc.Log = &HckioTestLogger{t: t}
	return hc
}

// K8sNotFoundError represents a not found error
type K8sNotFoundError struct{}

// Error returns the error message
func (e *K8sNotFoundError) Error() string {
	return "Not FOund"
}

// Status returns the status
func (e *K8sNotFoundError) Status() metav1.Status {
	return metav1.Status{
		Reason: metav1.StatusReasonNotFound,
	}
}

// HckioTestLogger is a healthchecksio.Logger that prints through a testing.T object
type HckioTestLogger struct {
	t *testing.T
}

// Debugf logs debug messages
func (l *HckioTestLogger) Debugf(format string, args ...interface{}) {
	l.t.Logf("[DEBUG] "+format, args...)
}

// Infof logs debug messages
func (l *HckioTestLogger) Infof(format string, args ...interface{}) {
	l.t.Logf("[INFO] "+format, args...)
}

// Errorf logs debug messages
func (l *HckioTestLogger) Errorf(format string, args ...interface{}) {
	l.t.Logf("[ERROR] "+format, args...)
}

// LogrTestLogger is a logr.Logger that prints through a testing.T object
// Only error logs will have any effect.
type LogrTestLogger struct {
	T *testing.T
}

// Info logs error messages
func (log LogrTestLogger) Info(msg string, args ...interface{}) {
	log.T.Logf("[INFO] "+msg, args...)
}

// Enabled returns true
func (log LogrTestLogger) Enabled() bool {
	return true
}

// Error logs error messages
func (log LogrTestLogger) Error(err error, msg string, args ...interface{}) {
	a := make([]interface{}, 0)
	a = append(a, err)
	a = append(a, args...)
	log.T.Logf("[ERROR] %v: "+msg, a...)
}

// V returns a logger
func (log LogrTestLogger) V(v int) logr.InfoLogger {
	return log
}

// WithName returns a logger
func (log LogrTestLogger) WithName(_ string) logr.Logger {
	return log
}

// WithValues returs a logger
func (log LogrTestLogger) WithValues(_ ...interface{}) logr.Logger {
	return log
}
