module github.com/kristofferahl/healthchecksio-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/uuid v1.1.1 // indirect
	github.com/kristofferahl/go-healthchecksio v1.1.1-0.20191110102359-6a12e8c5676a
	github.com/mitchellh/hashstructure v1.0.0
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.5.0
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734 // indirect
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	sigs.k8s.io/controller-runtime v0.3.0
)
