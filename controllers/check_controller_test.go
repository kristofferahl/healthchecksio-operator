package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/gomega"

	healthchecksio "github.com/kristofferahl/go-healthchecksio"
	monitoringv1alpha1 "github.com/kristofferahl/healthchecksio-operator/api/v1alpha1"
	"github.com/kristofferahl/healthchecksio-operator/testutil"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func TestCheckController_RemoveString(t *testing.T) {
	// Arrange
	g := NewGomegaWithT(t)
	input := []string{"foo", "bar", "baz"}
	expected := []string{"foo", "baz"}

	// Act
	actual := removeString(input, "bar")

	// Assert
	g.Expect(actual).To(Equal(expected))
}

func TestCheckController_ContainsString(t *testing.T) {
	// Arrange
	g := NewGomegaWithT(t)
	input := []string{"foo", "bar", "baz"}

	// Act & assert
	g.Expect(containsString(input, "foo")).To(BeTrue())
	g.Expect(containsString(input, "bar")).To(BeTrue())
	g.Expect(containsString(input, "baz")).To(BeTrue())
	g.Expect(containsString(input, "bin")).To(BeFalse())
}

func TestCheckController_IsTargetChannel(t *testing.T) {
	// Arrange
	g := NewGomegaWithT(t)

	// Act & assert
	g.Expect(isTargetChannel(&healthchecksio.HealthcheckChannelResponse{Kind: "email"}, "", "email")).To(BeTrue())
	g.Expect(isTargetChannel(&healthchecksio.HealthcheckChannelResponse{Kind: "email"}, "", "foo")).To(BeFalse())
	g.Expect(isTargetChannel(&healthchecksio.HealthcheckChannelResponse{Name: "mail-1", Kind: "email"}, "mail-1", "email")).To(BeTrue())
	g.Expect(isTargetChannel(&healthchecksio.HealthcheckChannelResponse{Name: "mail-2", Kind: "email"}, "mail-1", "email")).To(BeFalse())
}

func TestCheckController_IgnoreNotFound(t *testing.T) {
	// Arrange
	g := NewGomegaWithT(t)
	err := fmt.Errorf("foo bar")
	apiErr := &testutil.K8sNotFoundError{}

	// Act & assert
	g.Expect(ignoreNotFound(err)).To(Equal(err))
	g.Expect(ignoreNotFound(apiErr)).To(BeNil())
}

func TestCheckController_DeleteExternalResources_Found(t *testing.T) {
	// Arrange
	check := &monitoringv1alpha1.Check{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	ctx := NewCheckReconcilerTest(
		t,
		WithK8sObjects(check),
		WithHckioServerResponse(200, "{}"),
	)
	defer func() { ctx.Close() }()

	// Act
	err := ctx.Reconciler.deleteExternalResources(check)

	// Asert
	ctx.t.Expect(err).ToNot(HaveOccurred())
}

func TestCheckController_DeleteExternalResources_NotFound(t *testing.T) {
	// Arrange
	check := &monitoringv1alpha1.Check{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	ctx := NewCheckReconcilerTest(
		t,
		WithK8sObjects(check),
		WithHckioServerResponse(404, `{}`),
	)
	defer func() { ctx.Close() }()

	// Act
	err := ctx.Reconciler.deleteExternalResources(check)

	// Asert
	ctx.t.Expect(err).ToNot(HaveOccurred(), "check not found by hckio client, retrying won't help")
}

func TestCheckController_DeleteExternalResources_Error(t *testing.T) {
	// Arrange
	check := &monitoringv1alpha1.Check{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	ctx := NewCheckReconcilerTest(
		t,
		WithK8sObjects(check),
		WithHckioServerResponse(502, ""),
	)
	defer func() { ctx.Close() }()

	// Act
	err := ctx.Reconciler.deleteExternalResources(check)

	// Asert
	ctx.t.Expect(err).To(HaveOccurred(), "hckio client returns with error")
}

func TestCheckController_ConvertCheckToHealthcheck(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{})).
		To(Equal(healthchecksio.Healthcheck{Name: "/", Unique: []string{"name"}}))

	name := GenerateRandomString(5)
	namespace := GenerateRandomString(10)
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: namespace + "/" + name, Unique: []string{"name"}}))

	schedule := GenerateRandomString(5)
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Schedule: schedule,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: "/", Schedule: schedule, Unique: []string{"name"}}))

	timezone := GenerateRandomString(3)
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Timezone: timezone,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: "/", Timezone: timezone, Unique: []string{"name"}}))

	timeout := rand.Intn(1000)
	timeout32 := int32(timeout)
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Timeout: &timeout32,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: "/", Timeout: timeout, Unique: []string{"name"}}))

	grace := rand.Intn(1000)
	grace32 := int32(grace)
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			GracePeriod: &grace32,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: "/", Grace: grace, Unique: []string{"name"}}))

	tags := []string{"k8s", "ftw"}
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Tags: tags,
		},
	})).To(Equal(healthchecksio.Healthcheck{Name: "/", Tags: "k8s ftw", Unique: []string{"name"}}))

	channels := []string{"email-1", "sms-2"}
	g.Expect(convertToHealthcheck(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: channels,
		},
	}, channels...)).To(Equal(healthchecksio.Healthcheck{Name: "/", Channels: "email-1,sms-2", Unique: []string{"name"}}))
}

func TestCheckController_MatchChannelsToChannels(t *testing.T) {
	g := NewGomegaWithT(t)

	channels := []*healthchecksio.HealthcheckChannelResponse{
		&healthchecksio.HealthcheckChannelResponse{
			ID:   "1",
			Name: "email-1",
			Kind: "email",
		},
		&healthchecksio.HealthcheckChannelResponse{
			ID:   "2",
			Name: "email-2",
			Kind: "email",
		},
		&healthchecksio.HealthcheckChannelResponse{
			ID:   "3",
			Name: "sms-1",
			Kind: "sms",
		},
	}

	// Match no channels
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{}, channels...)).To(Equal([]string{}))
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"foo/bar"},
		},
	}, channels...)).To(Equal([]string{}))
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"foo"},
		},
	}, channels...)).To(Equal([]string{}))

	// Match all
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"*"},
		},
	}, channels...)).To(Equal([]string{"*"}))

	// Match kind/name channel
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"email/email-1"},
		},
	}, channels...)).To(Equal([]string{"1"}))

	// Match kind
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"email"},
		},
	}, channels...)).To(Equal([]string{"1", "2"}))
	g.Expect(matchTargetChannels(monitoringv1alpha1.Check{
		Spec: monitoringv1alpha1.CheckSpec{
			Channels: []string{"sms"},
		},
	}, channels...)).To(Equal([]string{"3"}))
}

func TestCheckController_UpdateCheckStatusFromHealtcheck_Changed(t *testing.T) {
	// Arrange
	now := time.Now()
	serverTime := metav1.NewTime(now)
	pings := rand.Intn(100)
	pings32 := int32(pings)

	r := NewCheckReconcilerTest(t, WithReconcilerClock(func() *metav1.Time {
		return &serverTime
	}))

	check := monitoringv1alpha1.Check{Status: monitoringv1alpha1.CheckStatus{}}
	id := GenerateRandomString(20)
	res := &healthchecksio.HealthcheckResponse{
		UpdateURL: "update/" + id,
		PingURL:   "ping/" + id,
		Status:    GenerateRandomString(5),
		LastPing:  now.Format(time.RFC3339),
		Pings:     pings,
	}

	// Act
	changed := r.Reconciler.updateCheckStatus(&check, *res)

	// Assert
	r.t.Expect(changed).To(BeTrue())
	r.t.Expect(check.Status.ID).To(Equal(id))
	r.t.Expect(check.Status.PingURL).To(Equal(res.PingURL))
	r.t.Expect(check.Status.LastUpdated).To(Equal(&serverTime))
	r.t.Expect(check.Status.Status).To(Equal(res.Status))
	r.t.Expect(check.Status.LastPing.Format(time.RFC3339)).To(Equal(serverTime.Format(time.RFC3339)))
	r.t.Expect(check.Status.Pings).To(Equal(&pings32))
}

func TestCheckController_UpdateCheckStatusFromHealtcheck_Unchanged(t *testing.T) {
	// Arrange
	now := time.Now()
	serverTime := metav1.NewTime(now)
	pings := rand.Intn(100)
	pings32 := int32(pings)

	r := NewCheckReconcilerTest(t, WithReconcilerClock(func() *metav1.Time {
		return &serverTime
	}))

	id := GenerateRandomString(20)
	res := &healthchecksio.HealthcheckResponse{
		UpdateURL: "update/" + id,
		PingURL:   "ping/" + id,
		Status:    GenerateRandomString(5),
		LastPing:  now.Format(time.RFC3339),
		Pings:     pings,
	}

	check := monitoringv1alpha1.Check{
		Status: monitoringv1alpha1.CheckStatus{
			ID:       id,
			PingURL:  res.PingURL,
			Status:   res.Status,
			LastPing: &serverTime,
			Pings:    &pings32,
		},
	}

	// Act
	changed := r.Reconciler.updateCheckStatus(&check, *res)

	// Assert
	r.t.Expect(changed).To(BeFalse())
}

func TestCheckController_CreateCheck(t *testing.T) {
	var (
		name        = "example"
		namespace   = "testnamespace"
		timeout     = int32(3600)
		graceperiod = int32(60)
		now         = time.Now()
		serverTime  = metav1.NewTime(now)
	)

	// Create a Reconciler test context
	ctx := NewCheckReconcilerTest(
		t,
		WithReconcilerClock(func() *metav1.Time { return &serverTime }),
		WithK8sObjects(&monitoringv1alpha1.Check{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: monitoringv1alpha1.CheckSpec{
				Timeout:     &timeout,
				GracePeriod: &graceperiod,
				Channels:    []string{"email"},
			},
		}),
		WithHckioServerResponse(200, `{
			"channels": [
				{
					"id": "4ec5a071-2d08-4baa-898a-eb4eb3cd6941",
					"name": "My Work Email",
					"kind": "email"
				},
				{
					"id": "746a083e-f542-4554-be1a-707ce16d3acc",
					"name": "My Phone",
					"kind": "sms"
				}
			]
		}`),
		WithHckioServerResponse(200, `{
			"channels": "",
			"grace": 60,
			"name": "testnamespace/exsample",
			"pause_url": "https://healthchecks.io/api/v1/checks/e71024f4-8537-4dd2-b742-ebe5a1685776/pause",
			"ping_url": "https://hc-ping.com/e71024f4-8537-4dd2-b742-ebe5a1685776",
			"timeout": 3600,
			"status": "new",
			"tags": "",
			"tz": "",
			"update_url": "https://healthchecks.io/api/v1/checks/e71024f4-8537-4dd2-b742-ebe5a1685776"
		}`),
	)
	req := NewReconcileRequest(name, namespace)

	// Act
	res, err := ctx.Reconciler.Reconcile(req)

	// Make sure reconcile had not errors and that we requeue after n time
	ctx.t.Expect(err).ToNot(HaveOccurred(), "expected no errors during reconcile")
	ctx.t.Expect(res).To(Equal(reconcile.Result{RequeueAfter: 1 * time.Minute}))

	// Make sure check can be found after reconcile
	check := &monitoringv1alpha1.Check{}
	err = ctx.Reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, check)
	ctx.t.Expect(err).ToNot(HaveOccurred())

	// Make sure finalizer was added
	ctx.t.Expect(len(check.ObjectMeta.Finalizers)).To(Equal(1), "expected single finalizer")
	ctx.t.Expect(check.ObjectMeta.Finalizers[0]).To(Equal(finalizerName))

	// Make sure an ID is set, or else the create probably failed silently
	ctx.t.Expect(check.Status.ID).To(Equal("e71024f4-8537-4dd2-b742-ebe5a1685776"))
}

func TestCheckController_DeleteCheck(t *testing.T) {
	var (
		name      = "example"
		namespace = "testnamespace"
		now       = metav1.Now()
	)

	// Create a Reconciler test context
	ctx := NewCheckReconcilerTest(
		t,
		WithK8sObjects(&monitoringv1alpha1.Check{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				DeletionTimestamp: &now,
				Finalizers: []string{
					finalizerName,
				},
			},
		}),
		WithHckioServerResponse(200, `{}`),
	)
	req := NewReconcileRequest(name, namespace)

	// Act
	res, err := ctx.Reconciler.Reconcile(req)

	// Make sure reconcile had not errors and that we requeue after n time
	ctx.t.Expect(err).ToNot(HaveOccurred(), "expected no errors during reconcile")
	ctx.t.Expect(res).To(Equal(reconcile.Result{}), "not expecting a requeue")

	// Make sure check can be found after reconcile
	check := &monitoringv1alpha1.Check{}
	err = ctx.Reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, check)
	ctx.t.Expect(err).ToNot(HaveOccurred())

	// Make sure finalizer was removed
	ctx.t.Expect(len(check.ObjectMeta.Finalizers)).To(Equal(0))
}

func GenerateRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type TestOptions struct {
	ReconcilerClock      func() *metav1.Time
	K8sObjects           []runtime.Object
	HckioAPIKey          string
	HckioBaseURL         string
	HckioServerResponses []*testutil.FakeServerResponse
}

type Option func(o *TestOptions) *TestOptions

func WithReconcilerClock(clockSource func() *metav1.Time) Option {
	return func(o *TestOptions) *TestOptions {
		o.ReconcilerClock = clockSource
		return o
	}
}

func WithHckioAPIKey(apiKey string) Option {
	return func(o *TestOptions) *TestOptions {
		o.HckioAPIKey = apiKey
		return o
	}
}

func WithHckioBaseURL(url string) Option {
	return func(o *TestOptions) *TestOptions {
		o.HckioBaseURL = url
		return o
	}
}

func WithHckioServerResponse(status int, body string) Option {
	return func(o *TestOptions) *TestOptions {
		o.HckioServerResponses = append(o.HckioServerResponses, &testutil.FakeServerResponse{
			StatusCode:   status,
			ResponseBody: body,
		})
		return o
	}
}

func WithK8sObjects(objs ...runtime.Object) Option {
	return func(o *TestOptions) *TestOptions {
		o.K8sObjects = objs
		return o
	}
}

// NewReconcileRequest creates a new request object
func NewReconcileRequest(name, namespace string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// NewCheckReconcilerWithT creates a new reconciler
func NewCheckReconcilerWithT(t *testing.T, o *TestOptions) *CheckReconciler {
	// Register known types
	s := scheme.Scheme
	s.AddKnownTypes(monitoringv1alpha1.GroupVersion, &monitoringv1alpha1.Check{})

	// Create a fake k8s client
	kc := fake.NewFakeClient(o.K8sObjects...)

	// Create a test client
	hc := testutil.NewTestHealthchecksioClient(t, o.HckioAPIKey, o.HckioBaseURL)

	return &CheckReconciler{
		Client: kc,
		Scheme: s,
		Log:    testutil.LogrTestLogger{T: t},
		Hckio:  hc,
		Clock: Clock{
			Source: o.ReconcilerClock,
		},
	}
}

type CheckReconcilerTestContext struct {
	t                    *GomegaWithT
	Reconciler           *CheckReconciler
	HealthchecksioServer *httptest.Server
}

func (c *CheckReconcilerTestContext) Close() {
	c.HealthchecksioServer.Close()
}

func NewCheckReconcilerTest(t *testing.T, opts ...Option) *CheckReconcilerTestContext {
	// Setup default test options
	st := time.Now()
	o := &TestOptions{
		HckioBaseURL:         "",
		HckioAPIKey:          "api-key",
		HckioServerResponses: make([]*testutil.FakeServerResponse, 0),
		K8sObjects:           make([]runtime.Object, 0),
		ReconcilerClock: func() *metav1.Time {
			now := metav1.NewTime(st)
			return &now
		},
	}

	// Apply all the options
	for _, opt := range opts {
		o = opt(o)
	}

	s := testutil.NewTestHealthchecksioServer(o.HckioServerResponses...)

	if o.HckioBaseURL == "" {
		o.HckioBaseURL = s.URL
	}

	r := NewCheckReconcilerWithT(t, o)
	g := NewGomegaWithT(t)

	return &CheckReconcilerTestContext{
		Reconciler:           r,
		HealthchecksioServer: s,
		t:                    g,
	}
}
